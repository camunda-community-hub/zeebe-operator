/*

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

// ZeebeClusterSpec defines the desired state of ZeebeCluster
type ZeebeClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	StatefulSetName      string `json:"statefulSetName,omitempty"`
	ClusterType          string `json:"clusterType,omitempty"`
	ServiceName          string `json:"serviceName,omitempty"`
	ElasticSearchEnabled bool   `json:"elasticSearchEnabled,omitempty"`
	ElasticSearchHost    string `json:"elasticSearchHost,omitempty"`
	ElasticSearchPort    int32  `json:"elasticSearchPort,omitempty"`
	KibanaEnabled        bool   `json:"kibanaEnabled,omitempty"`
	PrometheusEnabled    bool   `json:"prometheusEnabled,omitempty"`
	OperateEnabled       bool   `json:"operateEnabled,omitempty"`
	ZeebeHealthChecks    bool   `json:"zeebeHealthChecksEnabled,omitempty"`
}

// ZeebeClusterStatus defines the observed state of ZeebeCluster
type ZeebeClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ClusterName       string            `json:"clusterName"`
	StatusName        string            `json:"statusName"`
	ZeebeHealth       string            `json:"zeebeHealth"`
	ZeebeHealthReport string            `json:"zeebeHealthReport"`
	Conditions        []StatusCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,path=zeebeclusters,shortName=zb
// +kubebuilder:printcolumn:JSONPath=".status.statusName",name=Status,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.operateEnabled",name=Operate,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.elasticSearchEnabled",name=ElasticSearch,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.KibanaEnabled",name=Kibana,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.PrometheusEnabled",name=Prometheus,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.serviceName",name=Service,type=string
// +kubebuilder:printcolumn:JSONPath=".status.zeebeHealth",name=Zeebe Cluster Health,type=string
// ZeebeCluster is the Schema for the zeebeclusters API
type ZeebeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZeebeClusterSpec   `json:"spec,omitempty"`
	Status ZeebeClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ZeebeClusterList contains a list of ZeebeCluster
type ZeebeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZeebeCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZeebeCluster{}, &ZeebeClusterList{})
}

type ConditionStatus string

var (
	ConditionStatusHealthy   ConditionStatus = "Healthy"
	ConditionStatusUnhealthy ConditionStatus = "Unhealthy"
	ConditionStatusUnknown   ConditionStatus = "Unknown"
)

type StatusCondition struct {
	Type   string          `json:"type"`
	Status ConditionStatus `json:"status"`
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}
