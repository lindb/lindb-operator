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

// BrokerSpec defines the desired state of Broker
type BrokerSpec struct {
	// replicas is the desired number of replicas of the given Template.
	// These are replicas in the sense that they are instantiations of the
	// same Template, but individual replicas also have a consistent identity.
	// If unspecified, defaults to 1.
	// +optional
	// default: 1
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`

	// the image of broker, if not set, use the default image of ClusterSpec
	// +optional
	Image string `json:"image,omitempty"`

	// the port of broker http server
	// +required
	// default: 9000
	HttpPort int32 `json:"httpPort,omitempty"`

	// the port of broker grpc server
	// +required
	// default: 9001
	GrpcPort int32 `json:"grpcPort,omitempty"`

	// the monitor configuration
	// +optional
	// default: report-interval: 10s, url: http://broker1:9000/api/v1/write?db=_internal
	Monitor MonitorSpec `json:"monitor,omitempty"`
}

type LoggingSpec struct {
	// the directory of log
	// +required
	// default: /lindb/broker1
	Dir string `json:"dir,omitempty"`

	// the level of log
	// +required
	// default: debug
	Level string `json:"level,omitempty"`
}

type MonitorSpec struct {
	// the interval of monitor report
	// +required
	// default: 10s
	ReportInterval string `json:"reportInterval,omitempty"`

	// the url of monitor report
	// +required
	// default: http://broker1:9000/api/v1/write?db=_internal
	ReportUrl string `json:"url,omitempty"`
}

// BrokerStatus defines the observed state of Broker
type BrokerStatus struct {
	// the status of broker
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:printcolumn:JSONPath=".status.brokerStatus",name=BrokerStatus,type=string
	Status string `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Broker is the Schema for the brokers API
type Broker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BrokerSpec   `json:"spec,omitempty"`
	Status BrokerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BrokerList contains a list of Broker
type BrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Broker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Broker{}, &BrokerList{})
}
