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
		return !bqMock.HasDataset(dataset.Spec.Project, dataset.Spec.Name)
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

func TestBigqueryDatasetControllerCascadingDeleteOnlyChange(t *testing.T) {
	ctx := context.Background()

	dataset := naisv1.BigQueryDataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cascading-noop",
			Namespace: "default",
		},
		Spec: naisv1.BigQueryDatasetSpec{
			Name:            "test-cascading-noop-dataset",
			Description:     "test description",
			Location:        "europe-north1",
			Project:         "gcpproject",
			CascadingDelete: false,
		},
	}

	if err := k8sClient.Create(ctx, &dataset); err != nil {
		t.Fatalf("Failed to create dataset: %v", err)
	}

	// Wait for initial reconcile to complete (CreationTime set means onCreate ran)
	var err error
	gotten := eventually(100*time.Millisecond, 10, func() bool {
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, &dataset)
		return err == nil && dataset.Status.CreationTime > 0
	})
	if err != nil {
		t.Fatalf("Failed to get dataset after creation: %v", err)
	} else if !gotten {
		t.Fatal("Dataset never reached Created state")
	}

	// Record the update count after initial creation (onCreate uses Create, not Update)
	updateCountAfterCreate := bqMock.GetUpdateCount()

	// Change only CascadingDelete — this must not trigger a GCP Update call
	creationHash := dataset.Status.SynchronizationHash
	dataset.Spec.CascadingDelete = true
	if err := k8sClient.Update(ctx, &dataset); err != nil {
		t.Fatalf("Failed to update dataset with CascadingDelete=true: %v", err)
	}

	// Wait for the controller to reconcile and update the hash in status
	gotten = eventually(100*time.Millisecond, 15, func() bool {
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, &dataset)
		return err == nil && dataset.Status.SynchronizationHash != creationHash
	})
	if err != nil {
		t.Fatalf("Failed to get dataset after CascadingDelete change: %v", err)
	} else if !gotten {
		t.Fatal("SynchronizationHash was never updated after CascadingDelete-only change")
	}

	// The GCP Update must NOT have been called — only CascadingDelete changed, which
	// has no effect on the BigQuery dataset itself.
	if bqMock.GetUpdateCount() != updateCountAfterCreate {
		t.Errorf("expected no GCP Update call for a CascadingDelete-only change, but updateCount changed from %d to %d",
			updateCountAfterCreate, bqMock.GetUpdateCount())
	}
}

func TestMetadataEqual(t *testing.T) {
	makeDataset := func(name, ns, desc string, appLabel string, access []naisv1.DatasetAccess) naisv1.BigQueryDataset {
		labels := map[string]string{}
		if appLabel != "" {
			labels["app"] = appLabel
		}
		return naisv1.BigQueryDataset{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
			Spec: naisv1.BigQueryDatasetSpec{
				Name:        name,
				Description: desc,
				Access:      access,
			},
		}
	}

	baseAccess := []*bigquery.AccessEntry{
		{Role: "READER", EntityType: bigquery.UserEmailEntity, Entity: "user@example.com"},
	}
	baseExisting := &bigquery.DatasetMetadata{
		Name:        "ds",
		Description: "desc",
		Access:      baseAccess,
		Labels:      map[string]string{"team": "myns"},
	}
	baseDataset := makeDataset("ds", "myns", "desc", "", []naisv1.DatasetAccess{
		{Role: "READER", UserByEmail: "user@example.com"},
	})

	t.Run("equal when nothing changed", func(t *testing.T) {
		if !metadataEqual(baseDataset, baseExisting, baseAccess) {
			t.Error("expected equal")
		}
	})

	t.Run("not equal when name differs", func(t *testing.T) {
		other := *baseExisting
		other.Name = "other"
		if metadataEqual(baseDataset, &other, baseAccess) {
			t.Error("expected not equal")
		}
	})

	t.Run("not equal when description differs", func(t *testing.T) {
		other := *baseExisting
		other.Description = "changed"
		if metadataEqual(baseDataset, &other, baseAccess) {
			t.Error("expected not equal")
		}
	})

	t.Run("not equal when access differs", func(t *testing.T) {
		otherAccess := []*bigquery.AccessEntry{
			{Role: "WRITER", EntityType: bigquery.UserEmailEntity, Entity: "other@example.com"},
		}
		if metadataEqual(baseDataset, baseExisting, otherAccess) {
			t.Error("expected not equal")
		}
	})

	t.Run("equal with access in different order", func(t *testing.T) {
		twoAccess := []*bigquery.AccessEntry{
			{Role: "READER", EntityType: bigquery.UserEmailEntity, Entity: "a@example.com"},
			{Role: "WRITER", EntityType: bigquery.UserEmailEntity, Entity: "b@example.com"},
		}
		existing := &bigquery.DatasetMetadata{
			Name: "ds", Description: "desc",
			Access: []*bigquery.AccessEntry{
				{Role: "WRITER", EntityType: bigquery.UserEmailEntity, Entity: "b@example.com"},
				{Role: "READER", EntityType: bigquery.UserEmailEntity, Entity: "a@example.com"},
			},
			Labels: map[string]string{"team": "myns"},
		}
		if !metadataEqual(baseDataset, existing, twoAccess) {
			t.Error("expected equal (order-insensitive)")
		}
	})

	t.Run("not equal when team label differs", func(t *testing.T) {
		other := *baseExisting
		other.Labels = map[string]string{"team": "wrongns"}
		if metadataEqual(baseDataset, &other, baseAccess) {
			t.Error("expected not equal")
		}
	})

	t.Run("not equal when app label differs", func(t *testing.T) {
		ds := makeDataset("ds", "myns", "desc", "myapp", []naisv1.DatasetAccess{
			{Role: "READER", UserByEmail: "user@example.com"},
		})
		other := *baseExisting
		other.Labels = map[string]string{"team": "myns", "app": "otherapp"}
		if metadataEqual(ds, &other, baseAccess) {
			t.Error("expected not equal")
		}
	})

	t.Run("equal when app label matches", func(t *testing.T) {
		ds := makeDataset("ds", "myns", "desc", "myapp", []naisv1.DatasetAccess{
			{Role: "READER", UserByEmail: "user@example.com"},
		})
		other := *baseExisting
		other.Labels = map[string]string{"team": "myns", "app": "myapp"}
		if !metadataEqual(ds, &other, baseAccess) {
			t.Error("expected equal")
		}
	})
}

func TestAccessSetEqual(t *testing.T) {
	entry := func(role, entity string) *bigquery.AccessEntry {
		return &bigquery.AccessEntry{Role: bigquery.AccessRole(role), EntityType: bigquery.UserEmailEntity, Entity: entity}
	}

	t.Run("both nil", func(t *testing.T) {
		if !accessSetEqual(nil, nil) {
			t.Error("expected equal")
		}
	})
	t.Run("same single entry", func(t *testing.T) {
		a := []*bigquery.AccessEntry{entry("READER", "u@x.com")}
		b := []*bigquery.AccessEntry{entry("READER", "u@x.com")}
		if !accessSetEqual(a, b) {
			t.Error("expected equal")
		}
	})
	t.Run("different length", func(t *testing.T) {
		a := []*bigquery.AccessEntry{entry("READER", "u@x.com")}
		var b []*bigquery.AccessEntry
		if accessSetEqual(a, b) {
			t.Error("expected not equal")
		}
	})
	t.Run("different role", func(t *testing.T) {
		a := []*bigquery.AccessEntry{entry("READER", "u@x.com")}
		b := []*bigquery.AccessEntry{entry("WRITER", "u@x.com")}
		if accessSetEqual(a, b) {
			t.Error("expected not equal")
		}
	})
}

func eventually(delay time.Duration, maxIterations int, f func() bool) bool {
	for range maxIterations {
		if f() {
			return true
		}

		time.Sleep(delay)
	}
	return false
}
