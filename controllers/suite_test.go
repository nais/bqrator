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
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	naisv1 "github.com/nais/liberator/pkg/apis/google.nais.io/v1"
	"github.com/nais/liberator/pkg/crd"
	"google.golang.org/api/googleapi"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	bqMock    = &bqMocker{
		state: map[string]*bigquery.DatasetMetadata{},
	}
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logf.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{crd.YamlDirectory()},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		log.Fatal(err)
	}

	if err := naisv1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal(err)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatal(err)
	}

	setupBigQueryDatasetController(ctx)
	code := m.Run()

	if err := testEnv.Stop(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func setupBigQueryDatasetController(ctx context.Context) {
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		log.Fatal(err)
	}

	mgr := NewBigQueryDatasetReconciler(k8sManager.GetClient(), k8sManager.GetScheme(), bqMock)
	if err := mgr.SetupWithManager(k8sManager); err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := k8sManager.Start(ctx); err != nil {
			log.Fatal(err)
		}
	}()
}

type bqMocker struct {
	state map[string]*bigquery.DatasetMetadata
}

func (b *bqMocker) Get(ctx context.Context, projectID, name string) (*bigquery.DatasetMetadata, error) {
	fmt.Println("GET", projectID, name)
	dm, ok := b.state[projectID+"_"+name]
	if !ok {
		return nil, fmt.Errorf("dataset not found")
	}
	return dm, nil
}

func (b *bqMocker) Create(ctx context.Context, projectID string, dataset *bigquery.DatasetMetadata) error {
	fmt.Println("CREATE", projectID, dataset.Name)
	if _, ok := b.state[projectID+"_"+dataset.Name]; ok {
		return &googleapi.Error{
			Code:    409,
			Message: dataset.Name + " already exists",
		}
	}
	b.state[projectID+"_"+dataset.Name] = dataset
	return nil
}

func (b *bqMocker) Update(ctx context.Context, projectID, name string, dataset bigquery.DatasetMetadataToUpdate, etag string) error {
	fmt.Println("UPDATE", projectID, name)
	dm, ok := b.state[projectID+"_"+name]
	if !ok {
		return fmt.Errorf("dataset not found")
	}
	if dataset.Name != nil {
		dm.Name = dataset.Name.(string)
	}
	if dataset.Access != nil {
		dm.Access = dataset.Access
	}
	if dataset.Description != nil {
		dm.Description = dataset.Description.(string)
	}
	if dataset.DefaultEncryptionConfig != nil {
		dm.DefaultEncryptionConfig = dataset.DefaultEncryptionConfig
	}
	if dataset.DefaultTableExpiration != nil {
		dm.DefaultTableExpiration = dataset.DefaultTableExpiration.(time.Duration)
	}

	return nil
}

func (b *bqMocker) Delete(ctx context.Context, projectID, name string) error {
	fmt.Println("DELETE", projectID, name)
	if _, ok := b.state[projectID+"_"+name]; !ok {
		return fmt.Errorf("dataset not found")
	}
	delete(b.state, projectID+"_"+name)
	return nil
}
