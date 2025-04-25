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

package main

import (
	"context"
	"flag"
	"os"

	"github.com/nais/bqrator/controllers"
	google_nais_io_v1 "github.com/nais/liberator/pkg/apis/google.nais.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	//+kubebuilder:scaffold:imports

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"cloud.google.com/go/bigquery"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(google_nais_io_v1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0.0.0.0:8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", "0.0.0.0:8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.JSONEncoder()))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "73585216.nais.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	bqClient, err := bigquery.NewClient(context.Background(), "")
	if err != nil {
		setupLog.Error(err, "unable to create bigquery client")
		os.Exit(1)
	}

	bqMgr := controllers.NewBigQueryDatasetReconciler(mgr.GetClient(), mgr.GetScheme(), &bqWrapper{client: bqClient})
	if err = bqMgr.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BigQueryDataset")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

type bqWrapper struct {
	client *bigquery.Client
}

func (b *bqWrapper) Get(ctx context.Context, projectID, name string) (*bigquery.DatasetMetadata, error) {
	return b.client.DatasetInProject(projectID, name).Metadata(ctx)
}

func (b *bqWrapper) Create(ctx context.Context, projectID string, dataset *bigquery.DatasetMetadata) error {
	return b.client.DatasetInProject(projectID, dataset.Name).Create(ctx, dataset)
}

func (b *bqWrapper) Update(ctx context.Context, projectID, name string, dataset bigquery.DatasetMetadataToUpdate, etag string) error {
	_, err := b.client.DatasetInProject(projectID, name).Update(ctx, dataset, etag)
	return err
}

func (b *bqWrapper) Delete(ctx context.Context, projectID, name string) error {
	return b.client.DatasetInProject(projectID, name).Delete(ctx)
}
