/*
Copyright 2023.

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

type Image struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

type Main struct {
	Profile string `json:"profile"`
}

type Runes struct {
	Stage string `json:"stage"`
}

type Apis struct {
	Stage string `json:"stage"`
}

type Ingress struct {
	Paths []string `json:"paths"`
}

// RunesServiceSpec defines the desired state of RunesService
type RunesServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of RunesService. Edit runesservice_types.go to remove/update
	//Foo   string `json:"foo,omitempty"`

	Image Image `json:"image,omitempty"`
	Main  Main  `json:"main,omitempty"`
	Runes Runes `json:"runes,omitempty"`
	Apis  Apis  `json:"apis,omitempty"`

	VerifyAccessToken string  `json:"verifyAccessToken"`
	Ingress           Ingress `json:"ingress,omitempty"`

	Plugins []string `json:"plugins,omitempty"`

	HostAliases map[string]string `json:"hostAliases,omitempty"`
	ExtraEnv    map[string]string `json:"extraEnv,omitempty"`

	//Stage            string `json:"stage,omitempty"`
	//NameRunesService string `json:"nameRunesService,omitempty"`
	//ReplicaCount int32 `json:"replicaCount,omitempty"`
}

// RunesServiceStatus defines the observed state of RunesService
type RunesServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RunesService is the Schema for the runesservices API
type RunesService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunesServiceSpec   `json:"spec,omitempty"`
	Status RunesServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RunesServiceList contains a list of RunesService
type RunesServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunesService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RunesService{}, &RunesServiceList{})
}
