# bqrator

Bqrator is a tool for creating and managing BigQuery datasets. It is a
custom implementation to allow non-authoritative dataset resources to be created.

It will add and update permissions on the dataset according to the rules defined
in the resource.

## Development

This operator is built using [Kubebuilder](https://kubebuilder.io/).
The kustomize files in this repo is not used in production, but is left available
for reference.

## Verifying the bqrator image and its contents

The image is signed "keylessly" using [Sigstore cosign](https://github.com/sigstore/cosign).
To verify its authenticity run

```
cosign verify \
--certificate-identity "https://github.com/nais/bqrator/.github/workflows/build_and_push_image.yaml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
europe-north1-docker.pkg.dev/nais-io/nais/images/bqrator@sha256:<shasum>
```

The images are also attested with SBOMs in the [CycloneDX](https://cyclonedx.org/) format.
You can verify these by running

```
cosign verify-attestation --type cyclonedx \
--certificate-identity "https://github.com/nais/build_and_push_image.yaml/.github/workflows/build_and_push_image.yaml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
europe-north1-docker.pkg.dev/nais-io/nais/images/bqrator@sha256:<shasum>
```
