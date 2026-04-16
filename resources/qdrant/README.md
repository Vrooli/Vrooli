# Qdrant Resource

Managed Qdrant vector database runtime for embeddings and semantic search workflows.

## Intent

- Resource ID: `qdrant`
- Category: `storage`
- Driver: `docker-service`
- Portability tier: `full`

## Use Cases

- Store embeddings for semantic search and retrieval-augmented workflows.
- Provide a local vector database for scenarios that should not depend on hosted search infrastructure.
- Support AI pipelines that need nearest-neighbor search plus metadata filtering.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Qdrant-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

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

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for collection, embedding, or backup workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/qdrant/docs/OPERATIONS.md) as the architecture boundary for future migrations.
