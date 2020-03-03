/*
deployment.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

package controllers

import (
	"context"
	"path"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	kwiteReady string = "kwiteready"
	kwiteAlive string = "kwitealive"
)

// Create, initialize and return a new Deployent.
func (r *KwiteReconciler) getDeployment(req ctrl.Request, log logr.Logger) (*appsv1.Deployment, error) {
	replicas := int32(r.kwite.Spec.MinReplicas)
	lbls := getLabelSelector(req)
	matchLabels := metav1.LabelSelector{MatchLabels: getLabelSelector(req)}

	var ips []corev1.LocalObjectReference
	if len(r.kwite.Spec.ImagePullSecrets) > 0 {
		ips = r.kwite.Spec.ImagePullSecrets
	}

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
							Name:  kwiteName,
							Image: r.kwite.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          kwiteName,
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
										Path: path.Join(r.kwite.Spec.Url, kwiteAlive),
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
										Path: path.Join(r.kwite.Spec.Url, kwiteAlive),
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
										Path: path.Join(r.kwite.Spec.Url, kwiteReady),
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
					ImagePullSecrets: ips,
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

func (r *KwiteReconciler) updateDeploymentStatus(d *appsv1.Deployment, log logr.Logger) {
	r.kwite.Status.ReadyReplicas = int(d.Status.ReadyReplicas)
	r.kwite.Status.Ready = d.Status.ReadyReplicas == int32(r.kwite.Spec.MinReplicas)
}

// Reconcile the Deployment cluster state.
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
		r.updateDeploymentStatus(dep, log)

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
