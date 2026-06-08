# Agent Instructions

## Project

bqrator — Kubernetes operator that creates and manages BigQuery datasets as non-authoritative resources. Built with controller-runtime (Kubebuilder).

## Commands

```bash
# Verify code (ALWAYS run before committing)
mise run check        # fmt + staticcheck + vulncheck + deadcode + vet + helm-lint

# Format
mise run fmt          # gofumpt + go fix

# Test
mise run test         # envtest-backed controller tests

# Generate CRDs and deepcopy
mise run generate

# Build
mise run local:build
```

## Code Style

- Formatter: `gofumpt` (not `gofmt`)
- Linters: staticcheck, gosec, govulncheck, deadcode, go vet
- Helm charts must pass `helm lint --strict`

## Architecture Rules

- Use dependency injection via interfaces for external clients (see `BigQueryWrapper` pattern in `controllers/bigquery_wrapper.go`)
- Use Kubernetes Finalizers for external resource cleanup (finalizer: `bqrator.nais.io/finalizer`)
- Use structured logging via `sigs.k8s.io/controller-runtime/pkg/log` with context
- Return `ctrl.Result{}, err` from reconcilers — never panic on reconcile errors
- CRD types come from `github.com/nais/liberator`

## Key Files

| Path                                        | Purpose                                 |
|---------------------------------------------|-----------------------------------------|
| `main.go`                                   | Controller-runtime manager setup and DI |
| `controllers/bigquerydataset_controller.go` | Core reconciliation logic               |
| `controllers/bigquery_wrapper.go`           | GCP BigQuery client interface layer     |
| `pkg/metrics/`                              | Prometheus metrics                      |
| `charts/`                                   | Helm deployment chart                   |

## Do NOT

- Add direct GCP client calls without going through the wrapper interface
- Use `panic` or `os.Exit` in reconciliation paths
- Skip `mise run check` before declaring work complete
- Modify CRD types here (they live in `github.com/nais/liberator`)
