# Unstructured.io Resource

Managed Unstructured API runtime for document partitioning and extraction workflows.

## Intent

- Resource ID: `unstructured-io`
- Category: `ai`
- Driver: `docker-service`
- Host requirement: Docker/Docker Desktop with Linux containers
- Runtime image: pinned `linux/amd64` only

## Use Cases

- Convert documents into structured chunks for LLM and retrieval workflows.
- Run OCR-backed extraction locally for sensitive or internal documents.
- Provide a shared document-processing API across scenarios and pipelines.

## Architecture

This resource uses the `docker-service` structure. The pinned image is
approximately 9.81 GB and publishes only a Linux amd64 manifest. Linux amd64
is the validated host path; macOS amd64 and Windows amd64 remain conditional
Docker Desktop routes. Arm64 is not claimed because the image has no arm64
manifest. A portability tier is intentionally not authored here; it is a
derived deployability verdict.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Unstructured-specific Go logic when the manifest and shared control plane are not enough.
- `cli/internal/unstructured` owns typed health and document-processing requests.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Unstructured-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to Unstructured
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer Unstructured status interpretation
- `cli/internal/health`: Unstructured-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install unstructured-io

# Check status through the shared control plane
resource-unstructured-io status

# Typed document-processing commands
resource-unstructured-io health
resource-unstructured-io formats
resource-unstructured-io process --input document.pdf --output elements.json
```

Default endpoint:

- API: `http://localhost:11450`

## Readiness and operations

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for document processing workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Resource-local shell workflows are retired. Use the typed CLI or the shared resource control plane.
- `resource-unstructured-io health` is the readiness probe. It calls the
  service health endpoint and completes a small text partition request, so a
  running container that cannot serve the primary capability is unhealthy.
- The measured host run used 1.373 GiB (~1,404 MiB) resident memory during a
  `hi_res` scan parse, 6.12 seconds wall-clock, and a 9.81 GB image pull. The
  manifest rounds that observation to a 1,536 MiB requirement and keeps two
  CPU cores as the measured operating requirement.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/unstructured-io/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
