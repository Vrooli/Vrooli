# Qdrant Resource

Managed Qdrant vector database runtime for embeddings and semantic search workflows.

## Intent

- Resource ID: `qdrant`
- Category: `storage`
- Driver: `managed-service`
- Portability tier: `full`

## Use Cases

- Store embeddings for semantic search and retrieval-augmented workflows.
- Provide a local vector database for scenarios that should not depend on hosted search infrastructure.
- Support AI pipelines that need nearest-neighbor search plus metadata filtering.

## Architecture

This resource uses the native, checksum-verified Qdrant artifact through the shared
`managed-service` lifecycle. Docker is not required. The control plane defaults to
`managed-shared` on the control plane and `managed-private` in desktop bundles.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Qdrant-specific Go logic when the manifest and shared control plane are not enough.
- Historical `lib/` shell behavior has been retired; lifecycle and configuration
  behavior lives in the shared control plane and typed Go packages.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Qdrant-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to Qdrant
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer Qdrant status interpretation
- `cli/internal/health`: Qdrant-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install qdrant

# Check status through the shared control plane
resource-qdrant status
```

Connection defaults:

- REST: `http://localhost:6333`
- gRPC: `grpc://localhost:6334`

Readiness: `http://localhost:6333/readyz`.

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for collection, embedding, or backup workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Windows is supported at the managed-artifact and contract level; hardware-specific
  acceptance remains partial until a Windows runtime is available in CI.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/qdrant/docs/OPERATIONS.md) as the architecture boundary for future migrations.

## Maturity

M4 (2026-08-05): managed-service lifecycle, typed configuration, pinned native artifacts, readiness contract, and capability evidence are enforced by the fleet contract.
