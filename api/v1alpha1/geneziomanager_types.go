/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type GitConfigGenezio struct {
}

type GiteaProvider struct {
	URL                string `json:"url"`
	Username           string `json:"username"`
	Token              string `json:"token,omitempty"`
	TokenSecretKey     string `json:"tokenSecretKey,omitempty"`
	TokenSecretName    string `json:"tokenSecretName,omitempty"`
	Password           string `json:"password,omitempty"`
	PasswordSecretKey  string `json:"passwordSecretKey,omitempty"`
	PasswordSecretName string `json:"passwordSecretName,omitempty"`
}

type ContainerRegistryConfig struct {
	URL                string `json:"url"`
	Username           string `json:"username"`
	Password           string `json:"password,omitempty"`
	PasswordSecretKey  string `json:"passwordSecretKey,omitempty"`
	PasswordSecretName string `json:"passwordSecretName,omitempty"`
}

type GitConfig struct {
	Provider            string        `json:"provider"`
	DeployementRepoName string        `json:"deployementRepoName"`
	Gitea               GiteaProvider `json:"gitea,omitempty"`
	// More such as github, gitlab, bitbucket will be added here
}

type ArgoCDConfig struct {
	URL                string `json:"url,omitempty"`
	Username           string `json:"username,omitempty"`
	Password           string `json:"password,omitempty"`
	PasswordSecretKey  string `json:"passwordSecretKey,omitempty"`
	PasswordSecretName string `json:"passwordSecretName,omitempty"`
}

// GenezioManagerSpec defines the desired state of GenezioManager
type GenezioManagerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ArgoCDConfig            ArgoCDConfig            `json:"argocdConfig"`
	GitConfig               GitConfig               `json:"gitConfig"`
	ContainerRegistryConfig ContainerRegistryConfig `json:"containerRegistryConfig"`
	Region                  string                  `json:"region"`
	ContainerPort           int32                   `json:"containerPort"`
	ChartRepo               string                  `json:"chartRepo"`
	ChartRev                string                  `json:"chartRev"`
}

// GenezioManagerStatus defines the observed state of GenezioManager
type GenezioManagerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions store the status conditions of the Memcached instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GenezioManager is the Schema for the geneziomanagers API
// +kubebuilder:subresource:status
type GenezioManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GenezioManagerSpec   `json:"spec,omitempty"`
	Status GenezioManagerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GenezioManagerList contains a list of GenezioManager
type GenezioManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GenezioManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GenezioManager{}, &GenezioManagerList{})
}
