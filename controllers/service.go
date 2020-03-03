/*
service.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

package controllers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
)

// Reconcile the Horizontal Pod Autoscaler cluster state.
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
					Name:     kwiteName + "-ext",
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

// Determine and return the fqdn for the kwite
func (r *KwiteReconciler) getKwiteFqdn(key string, svc *corev1.Service, log logr.Logger) string {
	hn := key
	ips, err := net.LookupIP(hn)
	if err != nil {
		log.Error(err, "Failed address lookup kwite hostname "+hn)
		return hn
	}

	for _, ip := range ips {
		v4 := ip.To4()
		if v4 == nil {
			continue
		}

		// marshal the ip address as string-able
		if ipaddr, err := ip.MarshalText(); err != nil {
			log.Error(err, "Failed to marshall kwite address to string-able type.")
		} else {
			s := string(ipaddr)
			if hosts, err := net.LookupAddr(s); err != nil {
				log.Error(err, "Failed to obtain fqdn for "+s)
			} else {
				// the first address is the fqdn; don't want the trailing dot
				hn = ""
				for _, h := range hosts {
					log.Info("Found fqdn: " + h)
					if len(h) > len(hn) {
						hn = h
					}
				}
				hn = strings.TrimSuffix(hn, ".")
				hn = net.JoinHostPort(hn, strconv.Itoa(int(svc.Spec.Ports[0].Port)))
				log.Info("Finalized on fqdn: " + hn)
			}
		}
		break
	}

	return hn
}

// Build and return the rewrite (key) for the recncile request.
func (r *KwiteReconciler) getServiceHostName(req ctrl.Request) string {
	// note we use the req, which is really the Kwite and not the Service.
	// The name and namespace are the same, given the k8s resource generation.
	return fmt.Sprintf("%s.%s", req.Name, req.Namespace)
}

func (r *KwiteReconciler) updateAddressStatus(req ctrl.Request, svc *corev1.Service, log logr.Logger) {
	r.kwite.Status.Address = r.getKwiteFqdn(r.getServiceHostName(req), svc, log)
}

// Reconcile the Service cluster state.
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

		// Update observed status first
		r.updateAddressStatus(req, svc, log)

		if r.kwite.Spec.Port != iVal {
			svc.Spec.Ports[0].Port = int32(r.kwite.Spec.Port)
			doUpdate = true
		}
		if svc.Spec.Ports[0].TargetPort.IntValue() != int(kwitePort) {
			svc.Spec.Ports[0].TargetPort = intstr.IntOrString{
				StrVal: kwiteName,
			}
			doUpdate = true
		}
		if doUpdate {
			log.Info("Updating Service " + svc.GetName())
			err := r.Update(ctx, svc)
			if err != nil {
				log.Error(err, "Failed to update Service.")
				return false, err
			}
		}
	}

	return doUpdate, nil
}
