//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BigQueryDataset) DeepCopyInto(out *BigQueryDataset) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BigQueryDataset.
func (in *BigQueryDataset) DeepCopy() *BigQueryDataset {
	if in == nil {
		return nil
	}
	out := new(BigQueryDataset)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BigQueryDataset) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BigQueryDatasetList) DeepCopyInto(out *BigQueryDatasetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BigQueryDataset, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BigQueryDatasetList.
func (in *BigQueryDatasetList) DeepCopy() *BigQueryDatasetList {
	if in == nil {
		return nil
	}
	out := new(BigQueryDatasetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BigQueryDatasetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BigQueryDatasetSpec) DeepCopyInto(out *BigQueryDatasetSpec) {
	*out = *in
	if in.Access != nil {
		in, out := &in.Access, &out.Access
		*out = make([]DatasetAccess, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BigQueryDatasetSpec.
func (in *BigQueryDatasetSpec) DeepCopy() *BigQueryDatasetSpec {
	if in == nil {
		return nil
	}
	out := new(BigQueryDatasetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BigQueryDatasetStatus) DeepCopyInto(out *BigQueryDatasetStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BigQueryDatasetStatus.
func (in *BigQueryDatasetStatus) DeepCopy() *BigQueryDatasetStatus {
	if in == nil {
		return nil
	}
	out := new(BigQueryDatasetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatasetAccess) DeepCopyInto(out *DatasetAccess) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatasetAccess.
func (in *DatasetAccess) DeepCopy() *DatasetAccess {
	if in == nil {
		return nil
	}
	out := new(DatasetAccess)
	in.DeepCopyInto(out)
	return out
}
