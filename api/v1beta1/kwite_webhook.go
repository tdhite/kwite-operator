/*
kwite_webhook.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

package v1beta1

import (
	"fmt"
	"text/template"

	"github.com/tdhite/kwite/pkg/funcs"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var kwitelog = logf.Log.WithName("kwite-resource")

// valid Publish options
var publishOptions = []string{"ClusterIP", "Ingress", "LoadBalancer"}

func (r *Kwite) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-web-kwite-site-v1beta1-kwite,mutating=true,failurePolicy=fail,groups=web.kwite.site,resources=kwites,verbs=create;update,versions=v1beta1,name=mkwite.kwite.site

var _ webhook.Defaulter = &Kwite{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Kwite) Default() {
	kwitelog.Info("default", "name", r.Name)

	if r.Spec.Publish == "" {
		r.Spec.Publish = publishOptions[0]
	}

	if r.Spec.Url == "" {
		r.Spec.Url = "/"
	}

	if r.Spec.Image == "" {
		r.Spec.Image = "kwite:latest"
	}

	if r.Spec.Port == 0 {
		r.Spec.Port = 8080
	}

	if r.Spec.MaxReplicas <= 0 {
		r.Spec.MaxReplicas = 1
	}

	if r.Spec.MinReplicas <= 0 {
		r.Spec.MinReplicas = 0
	}

	if r.Spec.Memory == "" {
		r.Spec.Memory = "64Mi"
	}

	if r.Spec.CPU == "" {
		r.Spec.CPU = "200m"
	}

	if r.Spec.TargetCpu == 0 {
		r.Spec.TargetCpu = 80
	}

	if r.Spec.ImagePullSecrets == nil {
		r.Spec.ImagePullSecrets = []corev1.LocalObjectReference{}
	}

	if r.Spec.SecurityContext == nil {
		nonRoot := true
		readOnly := true
		allowEscalate := false
		var user int64 = 65534
		r.Spec.SecurityContext = &corev1.SecurityContext{
			RunAsNonRoot:             &nonRoot,
			ReadOnlyRootFilesystem:   &readOnly,
			AllowPrivilegeEscalation: &allowEscalate,
			RunAsUser:                &user,
			RunAsGroup:               &user,
		}
	}
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-web-kwite-site-v1beta1-kwite,mutating=false,failurePolicy=fail,groups=web.kwite.site,resources=kwites,versions=v1beta1,name=vkwite.kwite.site

var _ webhook.Validator = &Kwite{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Kwite) ValidateCreate() error {
	kwitelog.Info("validate create", "name", r.Name)

	return r.validateKwite()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Kwite) ValidateUpdate(old runtime.Object) error {
	kwitelog.Info("validate update", "name", r.Name)

	return r.validateKwite()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Kwite) ValidateDelete() error {
	kwitelog.Info("validate delete", "name", r.Name)

	return nil
}

// Validate the kwite object
func (r *Kwite) validateKwite() error {
	var allErrs field.ErrorList
	allErrs = r.validateKwiteName(allErrs)
	allErrs = r.validateKwiteSpec(allErrs)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "web.kwite.site", Kind: ControllerName},
		r.Name, allErrs)
}

// Validate that the Kwite name conforms to the rules on object fields
func (r *Kwite) validateKwiteName(allErrs field.ErrorList) field.ErrorList {
	if len(r.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength {
		// The kwite name length is 63 character like all Kubernetes objects
		// (which must fit in a DNS subdomain).
		fe := field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "must be no more than 63 characters")
		allErrs = append(allErrs, fe)
	}
	return allErrs
}

// Validate the Kwite Spec object
func (r *Kwite) validateKwiteSpec(allErrs field.ErrorList) field.ErrorList {
	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.

	fldPath := field.NewPath("spec")

	if fe := r.validateStringOptions(fldPath, "Publish", publishOptions, r.Spec.Publish); fe != nil {
		allErrs = append(allErrs, fe)
	}

	if fe := r.validateQuantity(fldPath, "CPU", r.Spec.CPU); fe != nil {
		allErrs = append(allErrs, fe)
	}

	if fe := r.validateQuantity(fldPath, "Memory", r.Spec.Memory); fe != nil {
		allErrs = append(allErrs, fe)
	}

	if fe := r.validateTemplate(fldPath, "template", &r.Spec.Template); fe != nil {
		allErrs = append(allErrs, fe)
	}

	if fe := r.validateTemplate(fldPath, "ready", &r.Spec.Ready); fe != nil {
		allErrs = append(allErrs, fe)
	}

	if fe := r.validateTemplate(fldPath, "alive", &r.Spec.Alive); fe != nil {
		allErrs = append(allErrs, fe)
	}

	return allErrs
}

// Validate that the quantity conforms to the rules on Kubernetes quantities.
func (r *Kwite) validateStringOptions(fldPath *field.Path, name string, options []string, value interface{}) *field.Error {
	for _, s := range options {
		if value.(string) == s {
			return nil
		}
	}
	msg := fmt.Sprintf("Invalid option %s specified for %s.", value.(string), fldPath.String())
	return field.Invalid(fldPath, value, msg)
}

// Validate that the quantity conforms to the rules on Kubernetes quantities.
func (r *Kwite) validateQuantity(fldPath *field.Path, name string, value interface{}) *field.Error {
	if _, err := resource.ParseQuantity(value.(string)); err != nil {
		return field.Invalid(fldPath, value, err.Error())
	}
	return nil
}

// Validate that the Template within the Spec parses successfully
func (r *Kwite) validateTemplate(fldPath *field.Path, name string, t *string) *field.Error {
	_, err := template.New(name).Funcs(funcs.TextTemplateFuncs()).Parse(string(*t))
	if err != nil {
		return field.Invalid(fldPath.Child(name), r.Name, err.Error())
	}
	return nil
}
