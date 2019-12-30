/*
groupversion_info.go

Copyright (c) 2019-2020 VMware, Inc.

SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
*/

// Package v1beta1 contains API Schema definitions for the web v1beta1 API group
// +kubebuilder:object:generate=true
// +groupName=web.kwite.site
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "web.kwite.site", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
