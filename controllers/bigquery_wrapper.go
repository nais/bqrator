package controllers

import (
	"context"

	"cloud.google.com/go/bigquery"
)

type BigQueryWrapper struct {
	client *bigquery.Client
}

var _ BigQuery = &BigQueryWrapper{}

func (b *BigQueryWrapper) Get(ctx context.Context, projectID, name string) (*bigquery.DatasetMetadata, error) {
	return b.client.DatasetInProject(projectID, name).Metadata(ctx)
}

func (b *BigQueryWrapper) Create(ctx context.Context, projectID string, dataset *bigquery.DatasetMetadata) error {
	return b.client.DatasetInProject(projectID, dataset.Name).Create(ctx, dataset)
}

func (b *BigQueryWrapper) Update(ctx context.Context, projectID, name string, dataset bigquery.DatasetMetadataToUpdate, etag string) error {
	_, err := b.client.DatasetInProject(projectID, name).Update(ctx, dataset, etag)
	return err
}

func (b *BigQueryWrapper) Delete(ctx context.Context, projectID, name string) error {
	return b.client.DatasetInProject(projectID, name).Delete(ctx)
}
