# AI Gateway proto sources

This folder is the canonical source for AI Gateway wire contracts.
Generated Go, TypeScript, and Python artifacts under `packages/proto/gen/`
are derived from these files and should not be edited by hand.

## What's here

- `v1/shared/health.proto` — wire contract for the `/health` endpoint.
  Mirrors `packages/api-core/health/health.go`'s `Response` and
  `DependencyStatus` types field-for-field with matching JSON names so
  the api-core handler chain produces JSON that decodes into the
  generated proto type without translation.

## Adding a new schema

Add new `.proto` files under `packages/proto/schemas/ai-gateway/v1/`
(or a `v1/<domain>/` subdirectory if the domain warrants its own
folder). Then run `cd packages/proto && make generate && make lint`
and commit the regenerated artifacts in `packages/proto/gen/`.

## Why this layout

`packages/proto/schemas/` is the canonical source of truth for every
scenario's wire contracts. Keeping AI Gateway contracts here lets API,
CLI, UI, docs, and Test Genie consume one provider-neutral schema source.
