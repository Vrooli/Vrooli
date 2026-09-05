# Unstructured.io Resource

Managed Unstructured API runtime for document partitioning and extraction workflows.

## Intent

- Resource ID: `unstructured-io`
- Category: `ai`
- Driver: `managed-service`
- Host requirement: Linux amd64 for the pinned runtime artifact
- Artifact source: digest-pinned upstream OCI filesystem, extracted and supervised without a Docker daemon

## Use Cases

- Convert documents into structured chunks for LLM and retrieval workflows.
- Run OCR-backed extraction locally for sensitive or internal documents.
- Provide a shared document-processing API across scenarios and pipelines.

## Architecture

This resource uses the `managed-service` structure. The pinned OCI image is
approximately 9.81 GB and publishes only a Linux amd64 manifest. Its
filesystem is materialized into the Vrooli artifact store and launched with
the image’s Python runtime directly; Docker is not part of start, stop, or
health-check lifecycle. macOS, Windows, and arm64 remain unsupported because
the upstream image publishes no native target for them.

The frozen runtime does not include `detectron2`. Fast and automatic
partitioning are supported; `hi_res` is explicitly partial and must report
that limitation instead of silently claiming OCR/layout parity with the full
Docker image.

- `resource.json` is the declarative authority for lifecycle, acquisition,
  artifact digests, ports, exports, health, and freshness metadata.
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

The Linux amd64 artifact is versioned `2025.09.11` and is authenticated by
the tree digest recorded in `resource.json`. The source image digest is also
recorded there; changing either requires fresh evidence.

## Readiness and operations

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for document processing workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Resource-local shell workflows are retired. Use the typed CLI or the shared resource control plane.
- `resource-unstructured-io health` is the readiness probe. It calls the
  service health endpoint and completes a small text partition request, so a
  running service that cannot serve the primary capability is unhealthy.
- The measured host run used 1.373 GiB (~1,404 MiB) resident memory during a
  `hi_res` scan parse, 6.12 seconds wall-clock, and a 9.81 GB image pull. The
  manifest rounds that observation to a 1,536 MiB requirement and keeps two
  CPU cores as the measured operating requirement.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/unstructured-io/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
