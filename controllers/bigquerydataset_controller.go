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

package controllers

import (
	"context"
	"os"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/nais/bqrator/pkg/metrics"
	google_nais_io_v1 "github.com/nais/liberator/pkg/apis/google.nais.io/v1"
	"google.golang.org/api/googleapi"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizer = "bqrator.nais.io/finalizer"

// BigQueryDatasetReconciler reconciles a BigQueryDataset object
type BigQueryDatasetReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	bigqueryClient BigQuery
}

func NewBigQueryDatasetReconciler(client client.Client, scheme *runtime.Scheme, bqClient BigQuery) *BigQueryDatasetReconciler {
	return &BigQueryDatasetReconciler{
		bigqueryClient: bqClient,
		Client:         client,
		Scheme:         scheme,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BigQueryDatasetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&google_nais_io_v1.BigQueryDataset{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=google.nais.io,resources=bigquerydatasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=google.nais.io,resources=bigquerydatasets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=google.nais.io,resources=bigquerydatasets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BigQueryDatasetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var dataset google_nais_io_v1.BigQueryDataset
	if err := r.Get(ctx, req.NamespacedName, &dataset); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch BigQueryDataset")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling BigQueryDataset", "name", dataset.Name)
	metrics.BigQueryDatasetProcessed.Inc()

	if !dataset.DeletionTimestamp.IsZero() {
		return r.onDelete(ctx, dataset)
	}

	if err := r.createOrUpdate(ctx, dataset); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BigQueryDatasetReconciler) createOrUpdate(ctx context.Context, dataset google_nais_io_v1.BigQueryDataset) error {
	log := log.FromContext(ctx)
	currentHash, err := dataset.Hash()
	if err != nil {
		log.Error(err, "unable to compute hash")
		return err
	}

	if !slices.Contains(dataset.Finalizers, finalizer) {
		controllerutil.AddFinalizer(&dataset, finalizer)
		if err := r.Update(ctx, &dataset); err != nil {
			log.Error(err, "unable to add finalizer")
			return err
		}
	}

	if dataset.Status.CreationTime == 0 {
		return r.onCreate(ctx, dataset, currentHash)
	} else if currentHash != dataset.Status.SynchronizationHash {
		return r.onUpdate(ctx, dataset, currentHash)
	}

	return nil
}

func (r *BigQueryDatasetReconciler) onUpdate(ctx context.Context, dataset google_nais_io_v1.BigQueryDataset, hash string) error {
	log := log.FromContext(ctx)

	existing, err := r.bigqueryClient.Get(ctx, dataset.Spec.Project, dataset.Spec.Name)
	if err != nil {
		log.Error(err, "Unable to fetch existing dataset")
		return err
	}

	access := createAccessList(dataset)
	existingAccess := removeDeletedServiceAccounts(existing.Access)
	for _, existingMember := range existingAccess {
		found := false
		for _, member := range access {
			// Entity will be empty string for view access, so we only compare on entity if it's not empty
			if existingMember.Entity == member.Entity && member.Entity != "" {
				found = true
				break
			}
		}
		if !found {
			access = append(access, existingMember)
		}
	}

	access = ensureBQratorOwner(access)

	metadata := bigquery.DatasetMetadataToUpdate{
		Name:        dataset.Spec.Name,
		Description: dataset.Spec.Description,
		Access:      access,
	}

	if metav1.HasLabel(dataset.ObjectMeta, "app") {
		metadata.SetLabel("app", dataset.GetLabels()["app"])
	}
	metadata.SetLabel("team", dataset.GetNamespace())

	err = r.bigqueryClient.Update(ctx, dataset.Spec.Project, dataset.Spec.Name, metadata, existing.ETag)
	if err != nil {
		log.Error(err, "unable to update dataset")
		return err
	}
	dataset.Status.LastModifiedTime = int(time.Now().Unix())
	meta.SetStatusCondition(&dataset.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time(metav1.NowMicro()),
		Reason:             "UpToDate",
		Message:            "The resource is up to date",
	})
	dataset.Status.SynchronizationHash = hash

	if err := r.Status().Update(ctx, &dataset); err != nil {
		log.Error(err, "unable to update status")
		return err
	}
	return nil
}

func (r *BigQueryDatasetReconciler) onDelete(ctx context.Context, dataset google_nais_io_v1.BigQueryDataset) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !slices.Contains(dataset.Finalizers, finalizer) {
		if err := r.Delete(ctx, &dataset); err != nil {
			log.Error(err, "unable to delete BigQueryDataset resource")
			return ctrl.Result{}, err
		}
		log.Info("Deleted BigQueryDataset", "name", dataset.Name)
		return ctrl.Result{}, nil
	}

	log.Info("Deleting BigQueryDataset", "name", dataset.Name)
	if dataset.Spec.CascadingDelete {
		if err := r.bigqueryClient.Delete(ctx, dataset.Spec.Project, dataset.Spec.Name); err != nil {
			meta.SetStatusCondition(&dataset.Status.Conditions, metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Time(metav1.NowMicro()),
				Reason:             "DeleteError",
				Message:            "Unable to delete from Google: " + err.Error(),
			})

			if err := r.Status().Update(ctx, &dataset); err != nil {
				log.Error(err, "unable to update status when deleting dataset")
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}

			if apierrors.IsNotFound(err) {
				log.Error(err, "unable to delete dataset")
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}

			log.Info("Ignoring deletion", "error", err)
			return ctrl.Result{}, nil
		}
	}

	controllerutil.RemoveFinalizer(&dataset, finalizer)
	if err := r.Update(ctx, &dataset); err != nil {
		log.Error(err, "unable to update BigQueryDataset")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BigQueryDatasetReconciler) onCreate(ctx context.Context, dataset google_nais_io_v1.BigQueryDataset, hash string) error {
	log := log.FromContext(ctx)

	dataset.Status.CreationTime = int(time.Now().Unix())
	dataset.Status.LastModifiedTime = dataset.Status.CreationTime

	meta.SetStatusCondition(&dataset.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time(metav1.NowMicro()),
		Reason:             "UpToDate",
		Message:            "The resource is up to date",
	})
	dataset.Status.SynchronizationHash = hash

	labels := map[string]string{
		"team": dataset.GetNamespace(),
	}

	if metav1.HasLabel(dataset.ObjectMeta, "app") {
		labels["app"] = dataset.GetLabels()["app"]
	}

	// TODO(thokra): Fields are optional, but we expect correct values as of now.
	err := r.bigqueryClient.Create(ctx, dataset.Spec.Project, &bigquery.DatasetMetadata{
		Name:        dataset.Spec.Name,
		Location:    dataset.Spec.Location,
		Description: dataset.Spec.Description,
		Access:      ensureBQratorOwner(createAccessList(dataset)),
		Labels:      labels,
	})
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 409 {
			log.Info("Dataset already exists")
			return r.onUpdate(ctx, dataset, hash)
		}
		log.Error(err, "unable to create dataset")
		return err
	}

	if err := r.Status().Update(ctx, &dataset); err != nil {
		log.Error(err, "unable to update status")
		return err
	}
	return nil
}

func createAccessList(dataset google_nais_io_v1.BigQueryDataset) []*bigquery.AccessEntry {
	var access []*bigquery.AccessEntry
	for _, member := range dataset.Spec.Access {
		// Add the operator user as owner by default
		access = append(access, &bigquery.AccessEntry{
			Role:       bigquery.AccessRole(member.Role),
			Entity:     member.UserByEmail,
			EntityType: bigquery.UserEmailEntity,
		})
	}
	return access
}

func removeDeletedServiceAccounts(accessList []*bigquery.AccessEntry) []*bigquery.AccessEntry {
	var newAccessList []*bigquery.AccessEntry
	for _, entry := range accessList {
		if !strings.HasPrefix(entry.Entity, "deleted:serviceAccount") {
			newAccessList = append(newAccessList, entry)
		}
	}

	return newAccessList
}

func ensureBQratorOwner(in []*bigquery.AccessEntry) []*bigquery.AccessEntry {
	bqratorEmail := os.Getenv("SA_ACCOUNT_EMAIL")
	if bqratorEmail == "" {
		return in
	}

	for _, entry := range in {
		if entry.Entity == bqratorEmail {
			return in
		}
	}

	return append(in, &bigquery.AccessEntry{
		Role:       bigquery.OwnerRole,
		Entity:     bqratorEmail,
		EntityType: bigquery.UserEmailEntity,
	})
}
