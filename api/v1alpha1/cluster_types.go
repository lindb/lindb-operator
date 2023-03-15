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

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// EtcdEndpoints is the etcd endpoints
	// +kubebuilder:validation:Required
	// +kubebuilder:default:="/lindb-cluster"
	EtcdNamespace string `json:"etcdNamespace,omitempty"`

	// EtcdEndpoints is the etcd endpoints
	// +kubebuilder:validation:Required
	// default: ["http://etcd:2379"]
	EtcdEndpoints []string `json:"etcdEndpoints,omitempty"`

	// Paused can be used to prevent controllers from processing the Cluster and all its associated objects.
	// +kubebuilder:validation:Optional
	// default: false
	Paused bool `json:"paused,omitempty"`

	// image is the image of the lindb cluster
	// +kubebuilder:validation:Required
	// +build:default:="lindb/lindb:latest"
	Image string `json:"image,omitempty"`

	// broker is the broker configuration
	Brokers BrokerSpec `json:"brokers,omitempty"`

	// storage is the storage configuration
	Storages StorageSpec `json:"storages,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// the status of cluster
	// +kubebuilder:validation:Optional
	// +kubebuilder:printcolumn:JSONPath=".status.clusterStatus",name=clusterStatus,type=string
	ClusterStatus string `json:"clusterStatus,omitempty"`

	// the status of brokers
	// +optional
	BrokerStatuses BrokerStatus `json:"brokerStatuses,omitempty"`

	// the status of storages
	// +optional
	StorageStatuses StorageStatus `json:"storageStatuses,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
