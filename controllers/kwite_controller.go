/*
kwite_controller.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

package controllers

import (
	"context"
	"fmt"
	"path"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1beta1 "github.com/tdhite/kwite-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	asv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	svcOwnerKey          = ".metadata.controller"
	kwitePortName string = "kwite"
	kwitePort     int32  = 8080
)

// KwiteReconciler reconciles a Kwite object
type KwiteReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	kwite  *webv1beta1.Kwite
}

func getLabelSelector(req ctrl.Request) map[string]string {
	m := make(map[string]string)
	m["kwite"] = req.Name
	return m
}

func (r *KwiteReconciler) getAllKwiteUrls(ctx context.Context, req ctrl.Request, log logr.Logger) ([]string, error) {
	var svcList corev1.ServiceList
	if err := r.List(ctx, &svcList, client.InNamespace(req.Namespace), client.MatchingFields{svcOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return nil, err
	}
	return nil, nil
}

// GetConfigMap creates a configmap for kwite deployments
func (r *KwiteReconciler) getConfigMap(req ctrl.Request, log logr.Logger) (*corev1.ConfigMap, error) {
	d := map[string]string{
		"url":      r.kwite.Spec.Url,
		"template": r.kwite.Spec.Template,
		"ready":    r.kwite.Spec.Ready,
		"alive":    r.kwite.Spec.Alive,
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			// Name the configmap identically to the owner kwite
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Data: d,
	}

	if err := ctrl.SetControllerReference(r.kwite, cm, r.Scheme); err != nil {
		log.Error(err, "Could not set kwite as owner of ConfigMap: ")
		return nil, err
	}

	return cm, nil
}

func (r *KwiteReconciler) reconcileConfigMap(ctx context.Context, req ctrl.Request, log logr.Logger) (bool, error) {
	cm := &corev1.ConfigMap{}

	if err := r.Get(ctx, req.NamespacedName, cm); err != nil {
		if apierrs.IsNotFound(err) {
			// Need to create a new ConfigMap for this kwite
			cm, err = r.getConfigMap(req, log)
			if err != nil {
				log.Error(err, "Failed to configure ConfigMap")
				return false, err
			}
			if err = r.Create(ctx, cm); err != nil {
				log.Error(err, "unable to create ConfigMap")
				return false, err
			}
		} else {
			log.Error(err, "unable to retrieve ConfigMap")
			return false, err
		}
	}

	// Check current state against the loaded cm.
	// However, if deleting, just leave it alone.
	doUpdate := false
	if cm.ObjectMeta.DeletionTimestamp.IsZero() {
		if r.kwite.Spec.Url != cm.Data["url"] {
			cm.Data["url"] = r.kwite.Spec.Url
			doUpdate = true
		}
		if r.kwite.Spec.Template != cm.Data["template"] {
			cm.Data["template"] = r.kwite.Spec.Template
			doUpdate = true
		}
		if r.kwite.Spec.Ready != cm.Data["ready"] {
			cm.Data["ready"] = r.kwite.Spec.Ready
			doUpdate = true
		}
		if r.kwite.Spec.Alive != cm.Data["alive"] {
			cm.Data["alive"] = r.kwite.Spec.Alive
			doUpdate = true
		}
		if doUpdate {
			log.Info("Updating ConfigMap " + cm.GetName())
			err := r.Update(ctx, cm)
			if err != nil {
				log.Error(err, "Failed to update configmap with new data.")
				return false, err
			}
		}
	}

	return doUpdate, nil
}

// GetDeployment creates the kwite deployment resource
func (r *KwiteReconciler) getDeployment(req ctrl.Request, log logr.Logger) (*appsv1.Deployment, error) {
	replicas := int32(r.kwite.Spec.MinReplicas)
	lbls := getLabelSelector(req)
	matchLabels := metav1.LabelSelector{MatchLabels: getLabelSelector(req)}

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Labels:    lbls,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &matchLabels,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kwite",
							Image: r.kwite.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          kwitePortName,
									ContainerPort: kwitePort,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(r.kwite.Spec.CPU),
									corev1.ResourceMemory: resource.MustParse(r.kwite.Spec.Memory),
								},
							},
							SecurityContext: r.kwite.Spec.SecurityContext,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "configs",
									MountPath: "/configs",
								},
							},
							StartupProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: path.Join(r.kwite.Spec.Url, "kwitealive"),
										Port: intstr.IntOrString{
											IntVal: kwitePort,
										},
									},
								},
								FailureThreshold: 5,
								PeriodSeconds:    1,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: path.Join(r.kwite.Spec.Url, "kwitealive"),
										Port: intstr.IntOrString{
											IntVal: kwitePort,
										},
									},
								},
								PeriodSeconds: 3,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: path.Join(r.kwite.Spec.Url, "kwiteready"),
										Port: intstr.IntOrString{
											IntVal: kwitePort,
										},
									},
								},
								PeriodSeconds: 3,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "configs",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: req.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(r.kwite, d, r.Scheme); err != nil {
		log.Error(err, "Could not set kwite as owner of Deployment: "+req.Name)
		return nil, err
	}
	return d, nil
}

// reconcileDeployment reconciles the kwite relative to the owned deployment
func (r *KwiteReconciler) reconcileDeployment(ctx context.Context, req ctrl.Request, log logr.Logger) (bool, error) {
	dep := &appsv1.Deployment{}

	if err := r.Get(ctx, req.NamespacedName, dep); err != nil {
		if apierrs.IsNotFound(err) {
			// No deployment, create it
			dep, err = r.getDeployment(req, log)
			if err != nil {
				log.Error(err, "failed to create deployment resource")
				return false, err
			}
			if err = r.Create(ctx, dep); err != nil {
				log.Error(err, "failed to create Deployment on the cluster")
				return false, err
			}
		} else {
			log.Error(err, "unable to retrieve Deployment in namespace "+req.Namespace)
			return false, err
		}
	}

	// Check current state against the loaded deployment and update as needed.
	// However, if deleting, just leave it alone.
	doUpdate := false
	if dep.ObjectMeta.DeletionTimestamp.IsZero() {
		// note: replicas get managed by HPA
		if r.kwite.Spec.Image != dep.Spec.Template.Spec.Containers[0].Image {
			dep.Spec.Template.Spec.Containers[0].Image = r.kwite.Spec.Image
			doUpdate = true
		}
		iVal := int(dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
		if r.kwite.Spec.Port != iVal {
			dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = int32(r.kwite.Spec.Port)
			doUpdate = true
		}
		if doUpdate {
			log.Info("Updating deployment " + dep.GetName())
			err := r.Update(ctx, dep)
			if err != nil {
				log.Error(err, "Failed to update Deployment.")
				return false, err
			}
		}
	}

	return doUpdate, nil
}

func (r *KwiteReconciler) getService(req ctrl.Request, log logr.Logger) (*corev1.Service, error) {
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     "ClusterIP",
			Selector: getLabelSelector(req),
			Ports: []corev1.ServicePort{
				{
					Name:     "kwite-ext",
					Protocol: "TCP",
					Port:     int32(r.kwite.Spec.Port),
					TargetPort: intstr.IntOrString{
						IntVal: kwitePort,
					},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(r.kwite, s, r.Scheme); err != nil {
		log.Error(err, "Could not set kwite as owner of Service: ", req.Name)
		return nil, err
	}

	return s, nil
}

func (r *KwiteReconciler) reconcileService(ctx context.Context, req ctrl.Request, log logr.Logger) (bool, error) {
	svc := &corev1.Service{}

	if err := r.Get(ctx, req.NamespacedName, svc); err != nil {
		if apierrs.IsNotFound(err) {
			// Need to create the service since it's not there
			svc, err = r.getService(req, log)
			if err != nil {
				log.Error(err, "failed to create Service resource")
				return false, err
			}
			if err := r.Create(ctx, svc); err != nil {
				log.Error(err, "failed to create Service on the cluster: ")
				return false, err
			}
		} else {
			log.Error(err, "unable to retrieve Service in namespace "+req.Namespace)
			return false, err
		}
	}

	// Check current state against the loaded service and update as needed.
	// However, if deleting, just leave it alone.
	doUpdate := false
	if svc.ObjectMeta.DeletionTimestamp.IsZero() {
		// check current state against the loaded deployment and update as needed
		iVal := int(svc.Spec.Ports[0].Port)
		if r.kwite.Spec.Port != iVal {
			svc.Spec.Ports[0].Port = int32(r.kwite.Spec.Port)
			doUpdate = true
		}
		if svc.Spec.Ports[0].TargetPort.IntValue() != int(kwitePort) {
			svc.Spec.Ports[0].TargetPort = intstr.IntOrString{
				StrVal: kwitePortName,
			}
			doUpdate = true
		}
		if doUpdate {
			log.Info("Updating Service " + svc.GetName())
			err := r.Update(ctx, svc)
			if err != nil {
				log.Error(err, "Failed to update Deployment.")
				return false, err
			}
		}
	}

	if *r.kwite.Spec.Public {
		// TODO: switch to ingress model and fixup
		r.kwite.Status.Address = fmt.Sprintf("%s.%s.svc.cluster.local",
			req.Name, req.Namespace)
	} else {
		r.kwite.Status.Address = fmt.Sprintf("%s.%s.svc.cluster.local",
			req.Name, req.Namespace)
	}

	r.kwite.Status.Port = int(svc.Spec.Ports[0].Port)

	return doUpdate, nil
}

func (r *KwiteReconciler) getHPA(req ctrl.Request, log logr.Logger) (*asv1.HorizontalPodAutoscaler, error) {
	minReplicas := int32(r.kwite.Spec.MinReplicas)
	maxReplicas := int32(r.kwite.Spec.MaxReplicas)
	targetCPU := int32(r.kwite.Spec.TargetCpu)

	hpa := &asv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: asv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: asv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       req.Name,
			},
			MinReplicas:                    &minReplicas,
			MaxReplicas:                    maxReplicas,
			TargetCPUUtilizationPercentage: &targetCPU,
		},
	}

	if err := ctrl.SetControllerReference(r.kwite, hpa, r.Scheme); err != nil {
		log.Error(err, "Could not set kwite as owner of HPA: ")
		return nil, err
	}

	return hpa, nil
}

func (r *KwiteReconciler) reconcileHPA(ctx context.Context, req ctrl.Request, log logr.Logger) (bool, error) {
	hpa := &asv1.HorizontalPodAutoscaler{}

	if err := r.Get(ctx, req.NamespacedName, hpa); err != nil {
		if apierrs.IsNotFound(err) {
			// Need to create the HPA since it's not there
			hpa, err = r.getHPA(req, log)
			if err != nil {
				log.Error(err, "failed to create HPA resource")
				return false, err
			}
			if err := r.Create(ctx, hpa); err != nil {
				log.Error(err, "failed to create HPA on the cluster: ")
				return false, err
			}
		} else {
			log.Error(err, "unable to list HPA items in namespace "+req.Namespace)
			return false, err
		}
	}

	// Check current state against the loaded HPA and update as needed.
	// However, if deleting, just leave it alone.
	doUpdate := false
	if hpa.ObjectMeta.DeletionTimestamp.IsZero() {
		iVal := int(*hpa.Spec.MinReplicas)
		if r.kwite.Spec.MinReplicas != iVal {
			r := int32(r.kwite.Spec.MinReplicas)
			hpa.Spec.MinReplicas = &r
			doUpdate = true
		}
		iVal = int(hpa.Spec.MaxReplicas)
		if r.kwite.Spec.MaxReplicas != iVal {
			hpa.Spec.MaxReplicas = int32(r.kwite.Spec.MaxReplicas)
			doUpdate = true
		}
		iVal = int(*hpa.Spec.TargetCPUUtilizationPercentage)
		if r.kwite.Spec.TargetCpu != iVal {
			t := int32(r.kwite.Spec.TargetCpu)
			hpa.Spec.TargetCPUUtilizationPercentage = &t
			doUpdate = true
		}
		if doUpdate {
			log.Info("Updating HPA " + hpa.GetName())
			err := r.Update(ctx, hpa)
			if err != nil {
				log.Error(err, "Failed to update Deployment.")
				return false, err
			}
		}
	}

	r.kwite.Status.CurrentReplicas = int(hpa.Status.CurrentReplicas)

	return doUpdate, nil
}

// +kubebuilder:rbac:groups=web.kwite.site,resources=kwites,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=web.kwite.site,resources=kwites/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

func (r *KwiteReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("kwite", req.NamespacedName)
	res := ctrl.Result{}

	// load the kwite object
	var kwite webv1beta1.Kwite
	if err := r.Get(ctx, req.NamespacedName, &kwite); err != nil {
		if apierrs.IsNotFound(err) {
			// might have been deleted or is simply not yet created
			return res, client.IgnoreNotFound(err)
		} else {
			// some real error occurred
			log.Error(err, "Unable to fetch kwite")
			return res, err
		}
	}

	// Cache this kwite for reconcilation ease
	r.kwite = &kwite

	// load our configmap and reconcile against it
	update, _ := r.reconcileConfigMap(ctx, req, log)

	// reconcile against the deployment
	u, _ := r.reconcileDeployment(ctx, req, log)
	update = u || update

	// reconcile against the service
	u, _ = r.reconcileService(ctx, req, log)
	update = u || update

	// reconcile against the hpa
	u, _ = r.reconcileHPA(ctx, req, log)
	update = u || update

	r.Status().Update(ctx, r.kwite)

	return res, nil
}

func isOwnerKwite(rawObj runtime.Object) []string {
	svc := rawObj.(*corev1.Service)
	owner := metav1.GetControllerOf(svc)
	if owner == nil {
		return nil
	}

	if owner.APIVersion != webv1beta1.GroupVersion.String() || owner.Kind != "Kwite" {
		return nil
	}

	return []string{owner.Name}
}

func (r *KwiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Scheme = mgr.GetScheme()

	if err := mgr.GetFieldIndexer().IndexField(&corev1.Service{}, svcOwnerKey,
		isOwnerKwite); err != nil {
		r.Log.Error(err, "Aborting setup.")
		return nil
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1beta1.Kwite{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
