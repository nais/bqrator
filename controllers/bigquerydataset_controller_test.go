package controllers

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/go-cmp/cmp"
	naisv1 "github.com/nais/liberator/pkg/apis/google.nais.io/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestBigqueryDatasetController(t *testing.T) {
	ctx := context.Background()

	dataset := naisv1.BigQueryDataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-set",
			Namespace: "default",
		},
		Spec: naisv1.BigQueryDatasetSpec{
			Name:        "test-dataset",
			Description: "test description",
			Location:    "europe-north1",
			Access: []naisv1.DatasetAccess{
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

	var err error
	gotten := eventually(100*time.Millisecond, 10, func() bool {
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, &dataset)
		return err == nil && dataset.Status.CreationTime > 0
	})
	if err != nil {
		t.Fatalf("Failed to get dataset: %v", err)
	} else if !gotten {
		t.Fatal("Never got the dataset from k8s")
	}

	if dataset.Status.CreationTime <= 0 {
		t.Errorf("CreationTime <= 0")
	}
	if dataset.Status.LastModifiedTime <= 0 {
		t.Errorf("LastModifiedTime <= 0")
	}
	status := meta.FindStatusCondition(dataset.Status.Conditions, "Ready")
	if status == nil {
		t.Errorf("expected 'READY' condition, but was not found")
	} else if status.Status != metav1.ConditionTrue {
		t.Errorf("expected status to be 'TRUE', got %q", status.Status)
	}
}

func TestBigqueryDatasetControllerAlreadyExistsInGCP(t *testing.T) {
	ctx := context.Background()

	bqMock.Create(ctx, "gcpproject", &bigquery.DatasetMetadata{
		Name:         "test-set-exists",
		Location:     "europe-north1",
		CreationTime: time.Now(),
	})
	dataset := naisv1.BigQueryDataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-set-exists",
			Namespace: "default",
		},
		Spec: naisv1.BigQueryDatasetSpec{
			Name:        "test-set-exists",
			Description: "test description",
			Location:    "europe-north1",
			Project:     "gcpproject",
		},
	}

	if err := k8sClient.Create(ctx, &dataset); err != nil {
		t.Fatalf("Failed to create dataset: %v", err)
	}

	var err error
	gotten := eventually(100*time.Millisecond, 10, func() bool {
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, &dataset)
		return err == nil && dataset.Status.CreationTime > 0
	})
	if err != nil {
		t.Fatalf("Failed to get dataset: %v", err)
	} else if !gotten {
		t.Fatal("Never got the dataset from k8s")
	}

	if dataset.Status.CreationTime <= 0 {
		t.Errorf("CreationTime <= 0")
	}
	if dataset.Status.LastModifiedTime <= 0 {
		t.Errorf("LastModifiedTime <= 0")
	}
	status := meta.FindStatusCondition(dataset.Status.Conditions, "Ready")
	if status == nil {
		t.Errorf("expected 'READY' condition, but was not found")
	} else if status.Status != metav1.ConditionTrue {
		t.Errorf("expected status to be 'TRUE', got %q", status.Status)
	}
}

func TestBigqueryDatasetControllerUpdate(t *testing.T) {
	ctx := context.Background()

	dataset := naisv1.BigQueryDataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-set-update",
			Namespace: "default",
		},
		Spec: naisv1.BigQueryDatasetSpec{
			Name:        "test-dataset-update",
			Description: "test description",
			Location:    "europe-north1",
			Access: []naisv1.DatasetAccess{
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

	// Wait 1s+ to ensure that creation second and modified second isn't equal
	time.Sleep(1500 * time.Millisecond)

	var err error
	gotten := eventually(100*time.Millisecond, 10, func() bool {
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, &dataset)
		return err == nil && dataset.Status.CreationTime > 0
	})
	if err != nil {
		t.Fatalf("Failed to get dataset: %v", err)
	} else if !gotten {
		t.Fatal("Never got the dataset from k8s")
	}

	dataset.Spec.Access = []naisv1.DatasetAccess{
		{
			Role:        "READER",
			UserByEmail: "mockuser1337@nav.no",
		},
	}
	if err := k8sClient.Update(ctx, &dataset); err != nil {
		t.Fatalf("Failed to update dataset: %v", err)
	}

	gotten = eventually(100*time.Millisecond, 15, func() bool {
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, &dataset)
		return err == nil && dataset.Status.LastModifiedTime > dataset.Status.CreationTime
	})
	if err != nil {
		t.Fatalf("Failed to get updated dataset: %v", err)
	} else if !gotten {
		t.Fatal("Never got the updated dataset from k8s")
	}

	metadata, err := bqMock.Get(ctx, dataset.Spec.Project, dataset.Spec.Name)
	if err != nil {
		t.Fatal(err)
	}

	expected := []*bigquery.AccessEntry{
		{
			Role:       "READER",
			EntityType: bigquery.UserEmailEntity,
			Entity:     "mockuser1337@nav.no",
		},
		{
			Role:       "WRITER",
			EntityType: bigquery.UserEmailEntity,
			Entity:     "test@helper.dev",
		},
	}

	if !cmp.Equal(metadata.Access, expected) {
		t.Error(cmp.Diff(metadata.Access, expected))
	}
}

func TestBigqueryDatasetControllerDelete(t *testing.T) {
	ctx := context.Background()

	dataset := naisv1.BigQueryDataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-set-delete",
			Namespace: "default",
		},
		Spec: naisv1.BigQueryDatasetSpec{
			Name:        "test-dataset-delete",
			Description: "test description",
			Location:    "europe-north1",
			Access: []naisv1.DatasetAccess{
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

	var err error
	gotten := eventually(100*time.Millisecond, 10, func() bool {
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, &dataset)
		return err == nil && dataset.Status.CreationTime > 0
	})
	if err != nil {
		t.Fatalf("Failed to get dataset: %v", err)
	} else if !gotten {
		t.Fatal("Never got the dataset from k8s")
	}

	if err := k8sClient.Delete(ctx, &dataset); err != nil {
		t.Fatalf("Failed to delete dataset: %v", err)
	}

	gotten = eventually(100*time.Millisecond, 10, func() bool {
		_, ok := bqMock.state[dataset.Spec.Project+"_"+dataset.Spec.Name]
		return !ok
	})
	if !gotten {
		t.Fatalf("Failed delete datset from state")
	}
}

func TestRemoveDeletedServiceAccounts(t *testing.T) {
	t.Run("removes deleted service accounts", func(t *testing.T) {
		existing := []*bigquery.AccessEntry{
			{Entity: "deleted:serviceAccount:user1"},
			{Entity: "serviceAccount:user1"},
			{Entity: "serviceAccount:user2"},
		}
		expected := []*bigquery.AccessEntry{
			{Entity: "serviceAccount:user1"},
			{Entity: "serviceAccount:user2"},
		}

		actual := removeDeletedServiceAccounts(existing)
		if !cmp.Equal(expected, actual) {
			t.Error(cmp.Diff(expected, actual))
		}
	})
	t.Run("doesn't do anything if no deleted service accounts", func(t *testing.T) {
		existing := []*bigquery.AccessEntry{
			{Entity: "serviceAccount:user1"},
			{Entity: "serviceAccount:user2"},
		}
		expected := []*bigquery.AccessEntry{
			{Entity: "serviceAccount:user1"},
			{Entity: "serviceAccount:user2"},
		}

		actual := removeDeletedServiceAccounts(existing)
		if !cmp.Equal(expected, actual) {
			t.Error(cmp.Diff(expected, actual))
		}
	})
	t.Run("handles nil", func(t *testing.T) {
		var existing []*bigquery.AccessEntry
		var expected []*bigquery.AccessEntry

		actual := removeDeletedServiceAccounts(existing)
		if !cmp.Equal(expected, actual) {
			t.Error(cmp.Diff(expected, actual))
		}
	})
}

func eventually(delay time.Duration, maxIterations int, f func() bool) bool {
	for i := 0; i < maxIterations; i++ {
		if f() {
			return true
		}

		time.Sleep(delay)
	}
	return false
}
