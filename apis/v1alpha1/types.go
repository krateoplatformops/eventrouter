package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A RegistrationSpec defines the desired state of a Registration.
type RegistrationSpec struct {
	ServiceName string `json:"serviceName"`
	Endpoint    string `json:"endpoint"`
}

// +kubebuilder:object:root=true

// A Registration registers a new eventrouter registration.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster
type Registration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RegistrationSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// RegistrationList contains a list of Registration.
type RegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Registration `json:"items"`
}
