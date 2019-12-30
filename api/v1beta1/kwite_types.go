/*
kwite_types.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KwiteSpec defines the desired state of Kwite
type KwiteSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinLength=0

	// The URL to handle in the kwite instances, default "/"
	Url string `json:"url"`

	// Whether the url is public (i.e., needs an ingress), default false
	// +optional
	Public *bool `json:"public,omitempty"`

	// container image to use for the http(s) server, default is kwite:latest
	// +optional
	Image string `json:"image"`

	// port on which to expose the Url, default is 8080
	// +optional
	Port int `json:"port,omitempty"`

	// +kubebuilder:validation:Minimum=1

	// The minimum number of page hander replicas, default is 1 (one)
	// +optional
	MinReplicas int `json:"minreplicas,omitempty"`

	// +kubebuilder:validation:Minimum=1

	// The maximum number of page hander replicas, default is 1 (one)
	// +optional
	MaxReplicas int `json:"maxreplicas,omitempty"`

	// Memory Resource request (e.g., "128Mi"), defaults to "64Mi"
	// +optional
	Memory string `json:"memory"`

	// CPU Resource request (e.g., "200m"), defaults to "200m"
	// +optional
	CPU string `json:"cpu"`

	// +kubebuilder:validation:Minimum=1

	// HorizontalPodAutoscaler CPU target utilization per pod, default is 80
	// +optional
	TargetCpu int `json:"targetcpu,omitempty"`

	// The security context for kwite instance Pods, default is no specified context
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// +kubebuilder:validation:MinLength=0

	// The template to execute for the kwite instances
	Template string `json:"template"`

	// +kubebuilder:validation:MinLength=0

	// The template to execute for the readiness probes
	Ready string `json:"ready"`

	// +kubebuilder:validation:MinLength=0

	// The template to execute for aliveness probes
	Alive string `json:"alive"`
}

// KwiteStatus defines the observed state of Kwite
type KwiteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The service address on which the URL is exposed
	Address string `json:"address,omitempty"`

	// The service Port on which the URL is exposed
	Port int `json:"port,omitempty"`

	// The minimum number of page hander replicas
	CurrentReplicas int `json:"minreplicas"`
}

// +kubebuilder:object:root=true

// Kwite is the Schema for the kwites API
type Kwite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KwiteSpec   `json:"spec,omitempty"`
	Status KwiteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KwiteList contains a list of Kwite
type KwiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kwite `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kwite{}, &KwiteList{})
}
