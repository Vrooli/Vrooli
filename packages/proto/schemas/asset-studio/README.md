# Asset Studio proto schemas

This directory is the source of truth for Asset Studio's Connect-RPC
contracts. Run `make generate` from `packages/proto/` after changing a schema
so the API, CLI, and UI receive matching generated types.

## What's here

- `v1/shared/health.proto` — wire contract for the health endpoint.
- `v1/studio/studio.proto` — identity-to-release artifact spine. Asset bytes
  are intentionally absent from consumer-facing messages; only safe metadata
  and governed references cross the boundary.

## Adding a schema

Add new `.proto` files under `v1/` (or a `v1/<domain>/` subdirectory when the
domain warrants it), then run `make generate && make lint` from
`packages/proto/`.

`packages/proto/schemas/` is the canonical source of truth for every
scenario's wire contracts.
