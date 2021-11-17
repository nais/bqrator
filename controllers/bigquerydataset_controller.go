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
	"time"

	naisiov1beta1 "nais/bqrator/api/v1beta1"

	"cloud.google.com/go/bigquery"
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
	bigqueryClient *bigquery.Client
}

func NewBigQueryDatasetReconciler(client client.Client, scheme *runtime.Scheme, bqClient *bigquery.Client) *BigQueryDatasetReconciler {
	return &BigQueryDatasetReconciler{
		bigqueryClient: bqClient,
		Client:         client,
		Scheme:         scheme,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BigQueryDatasetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1beta1.BigQueryDataset{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=nais.io.nais.io,resources=bigquerydatasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nais.io.nais.io,resources=bigquerydatasets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nais.io.nais.io,resources=bigquerydatasets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BigQueryDatasetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var dataset naisiov1beta1.BigQueryDataset
	if err := r.Get(ctx, req.NamespacedName, &dataset); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch BigQueryDataset")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling BigQueryDataset", "name", dataset.Name)

	if !dataset.DeletionTimestamp.IsZero() {
		return r.onDelete(ctx, dataset)
	}

	if err := r.createOrUpdate(ctx, dataset); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BigQueryDatasetReconciler) createOrUpdate(ctx context.Context, dataset naisiov1beta1.BigQueryDataset) error {
	log := log.FromContext(ctx)
	currentHash, err := dataset.Hash()
	if err != nil {
		log.Error(err, "unable to compute hash")
		return err
	}

	if !contains(dataset.Finalizers, finalizer) {
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

func (r *BigQueryDatasetReconciler) onUpdate(ctx context.Context, dataset naisiov1beta1.BigQueryDataset, hash string) error {
	log := log.FromContext(ctx)
	bqClient := r.bigqueryClient.DatasetInProject(dataset.Spec.Project, dataset.Spec.Name)

	existing, err := bqClient.Metadata(ctx)
	if err != nil {
		log.Error(err, "Unable to fetch existing dataset")
		return err
	}

	access := createAccessList(dataset)

	for _, existingMember := range existing.Access {
		found := false
		for _, member := range access {
			if existingMember.Entity == member.Entity {
				found = true
				break
			}
		}
		if !found {
			access = append(access, existingMember)
		}
	}

	_, err = bqClient.Update(ctx, bigquery.DatasetMetadataToUpdate{
		Name:        dataset.Spec.Name,
		Description: dataset.Spec.Description,
		Access:      access,
	}, existing.ETag)

	if err != nil {
		log.Error(err, "unable to update dataset")
		return err
	}
	dataset.Status.LastModifiedTime = int(time.Now().Unix())
	dataset.Status.Status = "READY"
	dataset.Status.SynchronizationHash = hash
	if err := r.Status().Update(ctx, &dataset); err != nil {
		log.Error(err, "unable to update status")
		return err
	}
	return nil
}

func (r *BigQueryDatasetReconciler) onDelete(ctx context.Context, dataset naisiov1beta1.BigQueryDataset) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	bqClient := r.bigqueryClient.DatasetInProject(dataset.Spec.Project, dataset.Spec.Name)

	if !contains(dataset.Finalizers, finalizer) {
		if err := r.Delete(ctx, &dataset); err != nil {
			log.Error(err, "unable to delete BigQueryDataset resource")
			return ctrl.Result{}, err
		}
		log.Info("Deleted BigQueryDataset", "name", dataset.Name)
		return ctrl.Result{}, nil
	}

	log.Info("Deleting BigQueryDataset", "name", dataset.Name)
	if dataset.Spec.CascadingDelete {
		if err := bqClient.Delete(ctx); err != nil {
			dataset.Status.Status = "ERROR"
			if err := r.Status().Update(ctx, &dataset); err != nil {
				log.Error(err, "unable to delete dataset")
				return ctrl.Result{RequeueAfter: 10 * time.Second}, err
			}
		}
	}

	controllerutil.RemoveFinalizer(&dataset, finalizer)
	if err := r.Update(ctx, &dataset); err != nil {
		log.Error(err, "unable to update BigQueryDataset")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BigQueryDatasetReconciler) onCreate(ctx context.Context, dataset naisiov1beta1.BigQueryDataset, hash string) error {

	log := log.FromContext(ctx)

	// TODO(thokra): Fields are optional, but we expect correct values as of now.
	err := r.bigqueryClient.DatasetInProject(dataset.Spec.Project, dataset.Spec.Name).Create(ctx, &bigquery.DatasetMetadata{
		Name:        dataset.Spec.Name,
		Location:    dataset.Spec.Location,
		Description: dataset.Spec.Description,
		Access:      createAccessList(dataset),
	})

	if err != nil {
		log.Error(err, "unable to create dataset")
		return err
	}

	dataset.Status.CreationTime = int(time.Now().Unix())
	dataset.Status.LastModifiedTime = dataset.Status.CreationTime
	dataset.Status.Status = "READY"
	dataset.Status.SynchronizationHash = hash
	if err := r.Status().Update(ctx, &dataset); err != nil {
		log.Error(err, "unable to update status")
		return err
	}
	return nil
}

func createAccessList(dataset naisiov1beta1.BigQueryDataset) []*bigquery.AccessEntry {
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

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
