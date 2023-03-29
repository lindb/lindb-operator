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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageSpec defines the desired state of Storage
type StorageSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:type=integer
	// default: 1
	Replicas *int32 `json:"replicas,omitempty"`

	// the image of storage, if not set, use the default image of ClusterSpec
	// +optional
	Image string `json:"image,omitempty"`

	// the port of storage grpc server
	// +required
	// default: 2891
	GrpcPort int32 `json:"grpcPort,omitempty"`

	// the port of storage http server
	// +required
	// default: 2892
	HttpPort int32 `json:"httpPort,omitempty"`

	// the monitor configuration
	// +optional
	// default: report-interval: 10s, url: http://broker1:9000/api/v1/write?db=_internal
	Monitor MonitorSpec `json:"monitor,omitempty"`

	// the resource configuration
	// +optional
	Resource corev1.ResourceList `json:"resource,omitempty"`
}

// StorageStatus defines the observed state of Storage
type StorageStatus struct {
	// the indicator of storage
	// +optional
	MyId int32 `json:"myid,omitempty"`

	// the status of storage
	// +kubebuilder:validation:Optional
	// +kubebuilder:printcolumn:JSONPath=".status.storageStatus",name=storageStatus,type=string
	Status string `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Storage is the Schema for the storages API
type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageSpec   `json:"spec,omitempty"`
	Status StorageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StorageList contains a list of Storage
type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Storage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Storage{}, &StorageList{})
}
