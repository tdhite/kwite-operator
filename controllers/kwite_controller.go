/*
kwite_controller.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1beta1 "github.com/tdhite/kwite-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
)

const (
	kwiteName string = "kwite"
	kwitePort int32  = 8080
)

// KwiteReconciler reconciles a Kwite object
type KwiteReconciler struct {
	client.Client
	Log          logr.Logger
	reconcileLog logr.Logger
	Scheme       *runtime.Scheme
	kwite        *webv1beta1.Kwite
}

func getLabelSelector(req ctrl.Request) map[string]string {
	m := make(map[string]string)
	m[kwiteName] = req.Name
	return m
}

// +kubebuilder:rbac:groups=web.kwite.site,resources=kwites,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=web.kwite.site,resources=kwites/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

func (r *KwiteReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	r.reconcileLog = r.Log.WithValues(kwiteName, req.NamespacedName)
	res := ctrl.Result{}

	// load the kwite object
	var kwite webv1beta1.Kwite
	if err := r.Get(ctx, req.NamespacedName, &kwite); err != nil {
		if apierrs.IsNotFound(err) {
			// might have been deleted or is simply not yet created
			return res, client.IgnoreNotFound(err)
		} else {
			// some real error occurred
			r.reconcileLog.Error(err, "Unable to fetch kwite")
			return res, err
		}
	}

	// Cache this kwite for reconcilation ease
	r.kwite = &kwite

	// get current status and setup to apply kwite url rewrites where appropriate
	update := r.updateDeploymentStatus(ctx, req) || r.updateHPAStatus(ctx, req) || r.updateServiceStatus(ctx, req)

	if update {
		if err := r.Status().Update(ctx, &kwite); err != nil {
			r.reconcileLog.Error(err, "Unable to update Kwite status")
			return ctrl.Result{}, err
		}
	}

	// reconcile against the various objects
	if err := r.reconcileDeployment(ctx, req); err != nil {
		r.reconcileLog.Error(err, "Failed to update Deployment for ", req.NamespacedName.String())
	}
	if err := r.reconcileService(ctx, req); err != nil {
		r.reconcileLog.Error(err, "Failed to update Service for ", req.NamespacedName.String())
	}
	if err := r.reconcileHPA(ctx, req); err != nil {
		r.reconcileLog.Error(err, "Failed to update HPA for ", req.NamespacedName.String())
	}
	if err := r.reconcileConfigMap(ctx, req); err != nil {
		r.reconcileLog.Error(err, "Failed to update ConfigMap for ", req.NamespacedName.String())
	}

	return res, nil
}

func isOwnerKwite(rawObj runtime.Object) []string {
	cm := rawObj.(*corev1.ConfigMap)
	owner := metav1.GetControllerOf(cm)

	if owner == nil {
		return nil
	} else if owner.APIVersion == webv1beta1.GroupVersion.String() && owner.Kind == webv1beta1.ControllerName {
		return []string{owner.Kind}
	} else {
		return nil
	}
}

func (r *KwiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Scheme = mgr.GetScheme()

	if err := mgr.GetFieldIndexer().IndexField(&corev1.ConfigMap{}, cmOwnerKey,
		isOwnerKwite); err != nil {
		r.reconcileLog.Error(err, "Aborting setup.")
		return nil
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1beta1.Kwite{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
