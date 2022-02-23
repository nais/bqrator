# bqrator

Bqrator is a tool for creating and managing BigQuery datasets. It is a
custom implementation to allow non-authoritative dataset resources to be created.

It will add and update permissions on the dataset according to the rules defined
in the resource.

## Development

This operator is built using [Kubebuilder](https://kubebuilder.io/).
The kustomize files in this repo is not used in production, but is left available
for reference.

The deploy is managed in nais-yaml and GCP permissions is managed in nais/gcp.
