package controllers

import (
	"context"

	"cloud.google.com/go/bigquery"
)

type BigQuery interface {
	Get(ctx context.Context, projectID, name string) (*bigquery.DatasetMetadata, error)
	Create(ctx context.Context, projectID string, dataset *bigquery.DatasetMetadata) error
	Update(ctx context.Context, projectID, name string, dataset bigquery.DatasetMetadataToUpdate, etag string) error
	Delete(ctx context.Context, projectID, name string) error
}

type BigQueryWrapper struct {
	Client *bigquery.Client
}

var _ BigQuery = &BigQueryWrapper{}

func (b *BigQueryWrapper) Get(ctx context.Context, projectID, name string) (*bigquery.DatasetMetadata, error) {
	return b.Client.DatasetInProject(projectID, name).Metadata(ctx)
}

func (b *BigQueryWrapper) Create(ctx context.Context, projectID string, dataset *bigquery.DatasetMetadata) error {
	return b.Client.DatasetInProject(projectID, dataset.Name).Create(ctx, dataset)
}

func (b *BigQueryWrapper) Update(ctx context.Context, projectID, name string, dataset bigquery.DatasetMetadataToUpdate, etag string) error {
	_, err := b.Client.DatasetInProject(projectID, name).Update(ctx, dataset, etag)
	return err
}

func (b *BigQueryWrapper) Delete(ctx context.Context, projectID, name string) error {
	return b.Client.DatasetInProject(projectID, name).Delete(ctx)
}
