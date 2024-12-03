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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BookStoreSpec defines the desired state of BookStore
type BookStoreSpec struct {
	Name      string        `json:"name"`
	Replicas  *int32        `json:"replicas"`
	Container ContainerSpec `json:"container,container"`
}

// BookStoreStatus defines the observed state of BookStore
type BookStoreStatus struct {
	State string `json:"state,omitempty"`
	Dep   bool   `json:"dep,omitempty"`
	Svc   bool   `json:"svc,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BookStore is the Schema for the bookstores API
type BookStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BookStoreSpec   `json:"spec"`
	Status BookStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BookStoreList contains a list of BookStore
type BookStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BookStore `json:"items"`
}
type ContainerSpec struct {
	Image string `json:"image,omitempty"`
	Port  int32  `json:"port,omitempty"`
}

func init() {
	SchemeBuilder.Register(&BookStore{}, &BookStoreList{})
}
