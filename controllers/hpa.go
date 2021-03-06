/*
hpa.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

package controllers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	asv1 "k8s.io/api/autoscaling/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
)

// Create, initialize and return a new Horizontal Pod Autoscaler.
func (r *KwiteReconciler) getHPA(req ctrl.Request) (*asv1.HorizontalPodAutoscaler, error) {
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
		r.reconcileLog.Error(err, "Could not set kwite as owner of HPA: ")
		return nil, err
	}

	return hpa, nil
}

func (r *KwiteReconciler) updateHPAStatus(ctx context.Context, req ctrl.Request) bool {
	hpa := &asv1.HorizontalPodAutoscaler{}
	doUpdate := false

	if err := r.Get(ctx, req.NamespacedName, hpa); err != nil {
		// no matter the error, no status update
		if apierrs.IsNotFound(err) {
			r.reconcileLog.Info("HPA does not exist for status update in namespace: " + req.NamespacedName.String())
		} else {
			r.reconcileLog.Error(err, "Failed HPA retrieve for status update in namespace: "+req.NamespacedName.String())
		}
	} else {
		r.kwite.Status.DesiredReplicas = int(hpa.Status.DesiredReplicas)
		doUpdate = true
	}
	return doUpdate
}

// Reconcile the Horizontal Pod Autoscaler cluster state.
func (r *KwiteReconciler) reconcileHPA(ctx context.Context, req ctrl.Request) error {
	hpa := &asv1.HorizontalPodAutoscaler{}

	if err := r.Get(ctx, req.NamespacedName, hpa); err != nil {
		if apierrs.IsNotFound(err) {
			// Need to create the HPA since it's not there
			hpa, err = r.getHPA(req)
			if err != nil {
				r.reconcileLog.Error(err, "failed to create HPA resource")
				return err
			}
			if err := r.Create(ctx, hpa); err != nil {
				r.reconcileLog.Error(err, "failed to create HPA on the cluster: ")
				return err
			}
		} else {
			r.reconcileLog.Error(err, "unable to list HPA items in namespace "+req.Namespace)
			return err
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
			r.reconcileLog.Info("Updating HPA " + hpa.GetName())
			err := r.Update(ctx, hpa)
			if err != nil {
				r.reconcileLog.Error(err, "Failed to update HPA.")
				return err
			}
		}
	}

	return nil
}
