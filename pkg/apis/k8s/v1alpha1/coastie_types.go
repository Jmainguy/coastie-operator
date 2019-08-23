package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CoastieSpec defines the desired state of Coastie
// +k8s:openapi-gen=true
type CoastieSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Tests          []string `json:"tests"`
	SlackChannelID string   `json:"slackchannelid"`
	SlackToken     string   `json:"slacktoken"`
	HostURL        string   `json:"hosturl"`
}

// CoastieStatus defines the observed state of Coastie
// +k8s:openapi-gen=true
type CoastieStatus struct {
	TestResults map[string]TestResult `json"testresults"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Coastie is the Schema for the coasties API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Coastie struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoastieSpec   `json:"spec,omitempty"`
	Status CoastieStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CoastieList contains a list of Coastie
type CoastieList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Coastie `json:"items"`
}

type TestResult struct {
	Status                string `json:"status,omitempty"`
	DaemonSetCreationTime string `json:"daemonsetcreationtime,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Coastie{}, &CoastieList{})
}
