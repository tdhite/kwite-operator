/*
configmap.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

package controllers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
)

const (
	cmOwnerKey = ".metadata.controller"
)

// Return all Kwite owned ConfigMaps.
func (r *KwiteReconciler) getAllConfigMaps(ctx context.Context, req ctrl.Request, log logr.Logger) (corev1.ConfigMapList, error) {
	var cmList corev1.ConfigMapList
	if err := r.List(ctx, &cmList, client.InNamespace(req.Namespace), client.MatchingFields{cmOwnerKey: req.Name}); err != nil {
		log.Error(err, "Unable to obtain child ConfigMap list.")
		return cmList, err
	}
	return cmList, nil
}

// Update the Kwite URL Map to storage.
func (r *KwiteReconciler) updateUrlMap(ctx context.Context, cm *corev1.ConfigMap, m map[string]string, log logr.Logger) {
	b, err := json.Marshal(m)
	if err != nil {
		log.Error(err, "Failed to convert rewrite map to JSON.")
		return
	}

	log.Info("Updating rewrite rules for ConfigMap " + cm.ObjectMeta.Name + "/" + cm.ObjectMeta.Namespace)
	cm.Data["rewrite"] = string(b)
	if err := r.Update(ctx, cm); err != nil {
		log.Error(err, "Failed to update reformed ConfigMap.")
	}
}

// Return a map from the provided JSON string.
func urlMapFromJson(s string) (map[string]string, error) {
	var m map[string]string
	if s == "" {
		return make(map[string]string), nil
	}

	b := []byte(s)
	if err := json.Unmarshal(b, &m); err != nil {
		log.Println(err)
		return nil, err
	} else {
		return m, nil
	}
}

// Fixup all Kwite owned ConfigMaps with the appropriate kwite
// scheme Url mapping .
func (r *KwiteReconciler) reformKwiteUrls(ctx context.Context, req ctrl.Request, log logr.Logger) {
	cmList, err := r.getAllConfigMaps(ctx, req, log)
	if err == nil {
		for _, cm := range cmList.Items {
			rewriteMap, err := urlMapFromJson(cm.Data["rewrite"])
			if err != nil {
				continue
			}
			key := r.getServiceHostName(req)
			if rewriteMap[key] != r.kwite.Status.Address {
				log.Info("Updating URL map entry for " + key + " to " + r.kwite.Status.Address)
				rewriteMap[key] = r.kwite.Status.Address
				r.updateUrlMap(ctx, &cm, rewriteMap, log)
			}
		}
	}
}

// Delete the url mapping for the quite from the ConfigMap.
func (r *KwiteReconciler) removeKwiteUrl(ctx context.Context, req ctrl.Request, log logr.Logger) {
	cmList, err := r.getAllConfigMaps(ctx, req, log)
	if err == nil {
		for _, cm := range cmList.Items {
			rewriteMap, err := urlMapFromJson(cm.Data["rewrite"])
			if err != nil {
				continue
			}
			key := r.getServiceHostName(req)
			log.Info("Updating URL map entry for " + key)
			delete(rewriteMap, key)
			r.updateUrlMap(ctx, &cm, rewriteMap, log)
		}
	}
}

// getConfigMap creates a configmap for kwite deployments
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

// Reconcile the ConfigMap's observed cluster state relative to desired state.
func (r *KwiteReconciler) reconcileConfigMap(ctx context.Context, req ctrl.Request, updateUrls bool, log logr.Logger) error {
	cm := &corev1.ConfigMap{}

	if err := r.Get(ctx, req.NamespacedName, cm); err != nil {
		if apierrs.IsNotFound(err) {
			// Need to create a new ConfigMap for this kwite
			cm, err = r.getConfigMap(req, log)
			if err != nil {
				log.Error(err, "Failed to configure ConfigMap")
				return err
			}
			if err = r.Create(ctx, cm); err != nil {
				log.Error(err, "unable to create ConfigMap")
				return err
			}
		} else {
			log.Error(err, "unable to retrieve ConfigMap")
			return err
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

		if updateUrls {
			r.reformKwiteUrls(ctx, req, log)
		}

		if doUpdate {
			log.Info("Updating ConfigMap " + cm.GetName())
			err := r.Update(ctx, cm)
			if err != nil {
				log.Error(err, "Failed to update ConfigMap.")
				return err
			}
		}
	}

	return nil
}
