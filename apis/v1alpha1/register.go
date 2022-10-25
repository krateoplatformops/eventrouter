package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "eventrouter.krateo.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

// Registration type metadata.
var (
	RegistrationKind             = reflect.TypeOf(Registration{}).Name()
	RegistrationGroupKind        = schema.GroupKind{Group: Group, Kind: RegistrationKind}.String()
	RegistrationKindAPIVersion   = RegistrationKind + "." + SchemeGroupVersion.String()
	RegistrationGroupVersionKind = SchemeGroupVersion.WithKind(RegistrationKind)
)

func init() {
	SchemeBuilder.Register(&Registration{}, &RegistrationList{})
}
