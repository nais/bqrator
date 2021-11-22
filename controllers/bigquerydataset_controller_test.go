package controllers

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/nais/bqrator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestBigqueryDatasetController(t *testing.T) {
	ctx := context.Background()

	dataset := v1beta1.BigQueryDataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-set",
			Namespace: "default",
		},
		Spec: v1beta1.BigQueryDatasetSpec{
			Name:        "test-dataset",
			Description: "test description",
			Location:    "europe-north1",
			Access: []v1beta1.DatasetAccess{
				{
					Role:        "WRITER",
					UserByEmail: "test@helper.dev",
				},
			},
			Project:         "gcpproject",
			CascadingDelete: true,
		},
	}

	if err := k8sClient.Create(ctx, &dataset); err != nil {
		t.Fatalf("Failed to create dataset: %v", err)
	}

	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, &dataset)
	if err != nil {
		t.Fatalf("Failed to get dataset: %v", err)
	}

	spew.Config.Dump(dataset.Status)
	if dataset.Status.CreationTime > 0 {
		t.Errorf("CreationTime < 0")
	}
	if dataset.Status.LastModifiedTime > 0 {
		t.Errorf("LastModifiedTime < 0")
	}
	if dataset.Status.Status != "READY" {
		t.Errorf("expected status to be 'READY', got %q", dataset.Status.Status)
	}
}
