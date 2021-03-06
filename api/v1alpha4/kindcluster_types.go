/*
Copyright 2021.

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

package v1alpha4

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

const (
	// ClusterFinalizer allows us to clean up the KIND cluster resource associated with KINDCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "kindcluster.infrastructure.cluster.x-k8s.io"
)

// KINDClusterSpec defines the desired state of KINDCluster
type KINDClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name represents the name of the KIND cluster
	// +required
	Name string `json:"name"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint *clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`
}

// KINDClusterStatus defines the observed state of KINDCluster
type KINDClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// +optional
	FailureReason string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KINDCluster is the Schema for the kindclusters API
type KINDCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KINDClusterSpec   `json:"spec,omitempty"`
	Status KINDClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KINDClusterList contains a list of KINDCluster
type KINDClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KINDCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KINDCluster{}, &KINDClusterList{})
}
