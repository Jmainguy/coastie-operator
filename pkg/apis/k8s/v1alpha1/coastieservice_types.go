package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CoastieServiceSpec defines the desired state of CoastieService
// +k8s:openapi-gen=true
type CoastieServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
    Tests   []string `json:"tests"`
}

// CoastieServiceStatus defines the observed state of CoastieService
// +k8s:openapi-gen=true
type CoastieServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
    Tests []Test `json"tests"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CoastieService is the Schema for the coastieservices API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type CoastieService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoastieServiceSpec   `json:"spec,omitempty"`
	Status CoastieServiceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CoastieServiceList contains a list of CoastieService
type CoastieServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoastieService `json:"items"`
}

type Test struct {
    Name    string `json:"name"`
    Status  string `json:"status"`
}


func init() {
	SchemeBuilder.Register(&CoastieService{}, &CoastieServiceList{})
}
