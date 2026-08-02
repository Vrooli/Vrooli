# Cleanup Manager Proto Sources

This folder is the canonical wire-contract source for cleanup-manager.
After editing these schemas, run `make generate` in `packages/proto/`
and commit the regenerated Go, Python, and TypeScript artifacts.

## What's here

- `v1/shared/health.proto` — wire contract for the `/health` endpoint.
  Mirrors `packages/api-core/health/health.go`'s `Response` and
  `DependencyStatus` types field-for-field with matching JSON names so
  the api-core handler chain produces JSON that decodes into the
  generated proto type without translation.

  The namespace follows the scenario convention used by existing entries in
  `packages/proto/schemas/`.
- `v1/cleanup/cleanup.proto` — CleanupService contract for provider
  catalog, policy profile, plan, apply, and audit operations.

## Adding a new schema

Add new `.proto` files under `packages/proto/schemas/cleanup-manager/v1/`
or a `v1/<domain>/` subdirectory if the domain warrants its own folder.
Then run `cd packages/proto && make generate && make lint` and commit the
regenerated artifacts in `packages/proto/gen/`.

## Why this layout

`packages/proto/schemas/` is the canonical source of truth for every
scenario's wire contracts. Cleanup Manager imports generated artifacts from
this directory through `packages/proto/gen/`, so schema changes must land
with generated code in the same change.
