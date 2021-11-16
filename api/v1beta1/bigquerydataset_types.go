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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DatasetAccess struct {
	Role string `json:"role,omitempty"`

	/* An email address of a user to grant access to. For example:
	fred@example.com. */
	UserByEmail string `json:"userByEmail,omitempty"`
}

// BigQueryDatasetSpec defines the desired state of BigQueryDataset
type BigQueryDatasetSpec struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Location    string          `json:"location,omitempty"`
	Access      []DatasetAccess `json:"access,omitempty"`
	Project     string          `json:"project,omitempty"`
}

// BigQueryDatasetStatus defines the observed state of BigQueryDataset
type BigQueryDatasetStatus struct {
	Status              string `json:"status,omitempty"`
	SynchronizationHash string `json:"synchronizationHash,omitempty"`
	CreationTime        int    `json:"creationTime,omitempty"`
	LastModifiedTime    int    `json:"lastModifiedTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BigQueryDataset is the Schema for the bigquerydatasets API
type BigQueryDataset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BigQueryDatasetSpec   `json:"spec,omitempty"`
	Status BigQueryDatasetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BigQueryDatasetList contains a list of BigQueryDataset
type BigQueryDatasetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BigQueryDataset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BigQueryDataset{}, &BigQueryDatasetList{})
}
