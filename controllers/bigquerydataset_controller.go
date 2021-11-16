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

//+kubebuilder:rbac:groups=nais.io.nais.io,resources=bigquerydatasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nais.io.nais.io,resources=bigquerydatasets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nais.io.nais.io,resources=bigquerydatasets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BigQueryDataset object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *BigQueryDatasetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var dataset naisiov1beta1.BigQueryDataset
	if err := r.Get(ctx, req.NamespacedName, &dataset); err != nil {
		log.Error(err, "unable to fetch BigQueryDataset")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling BigQueryDataset", "name", dataset.Name)

	currentHash, err := dataset.Hash()
	if err != nil {
		log.Error(err, "unable to compute hash")
		return ctrl.Result{}, err
	}

	if dataset.DeletionTimestamp.IsZero() {
		// Update operation
		if !contains(dataset.Finalizers, finalizer) {
			controllerutil.AddFinalizer(&dataset, finalizer)
			if err := r.Update(ctx, &dataset); err != nil {
				log.Error(err, "unable to add finalizer")
				return ctrl.Result{}, err
			}
		}

		if dataset.Status.CreationTime == 0 {
			var access []*bigquery.AccessEntry

			for _, member := range dataset.Spec.Access {
				// Add the operator user as owner by default
				access = append(access, &bigquery.AccessEntry{
					Role:       bigquery.AccessRole(member.Role),
					Entity:     member.UserByEmail,
					EntityType: bigquery.UserEmailEntity,
				})
			}

			// TODO(thokra): Fields are optional, but we expect correct values as of now.
			err := r.bigqueryClient.DatasetInProject(dataset.Spec.Project, dataset.Spec.Name).Create(ctx, &bigquery.DatasetMetadata{
				Name:        dataset.Spec.Name,
				Location:    dataset.Spec.Location,
				Description: dataset.Spec.Description,
				Access:      access,
			})
			if err != nil {
				log.Error(err, "unable to create dataset")
				return ctrl.Result{}, err
			}

			dataset.Status.CreationTime = int(time.Now().Unix())
			dataset.Status.LastModifiedTime = dataset.Status.CreationTime
			dataset.Status.Status = "READY"
			dataset.Status.SynchronizationHash = currentHash
			if err := r.Status().Update(ctx, &dataset); err != nil {
				log.Error(err, "unable to update status")
				return ctrl.Result{}, err
			}
		} else {
			if currentHash != dataset.Status.SynchronizationHash {
				// Update dataset
				log.Info("Do some updating")
			}
		}

	} else {
		// DELETE if finalizer is not present
		if contains(dataset.Finalizers, finalizer) {
			log.Info("Deleting BigQueryDataset", "name", dataset.Name)

			// TODO(thokra): Delete the dataset from BigQuery using the API.

			controllerutil.RemoveFinalizer(&dataset, finalizer)
			if err := r.Update(ctx, &dataset); err != nil {
				log.Error(err, "unable to update BigQueryDataset")
				return ctrl.Result{}, err
			}
		} else {
			if err := r.Delete(ctx, &dataset); err != nil {
				log.Error(err, "unable to delete BigQueryDataset")
				return ctrl.Result{}, err
			}
			log.Info("Deleted BigQueryDataset", "name", dataset.Name)
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BigQueryDatasetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1beta1.BigQueryDataset{}).
		Complete(r)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
