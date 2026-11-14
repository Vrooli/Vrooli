# Token Economy proto sources

This directory is the canonical source of truth for Token Economy wire
contracts. Generated Go, Python, and TypeScript artifacts live under
`packages/proto/gen/` and must be refreshed after source changes.

## What's here

- `v1/shared/health.proto` — wire contract for the `/health` endpoint.
  Mirrors `packages/api-core/health/health.go`'s `Response` and
  `DependencyStatus` types field-for-field with matching JSON names so
  the api-core handler chain produces JSON that decodes into the
  generated proto type without translation.

- `v1/shared/errors.proto` — common error envelope for deliberate REST edges.
- `v1/{mints,journal,grants,holders,earning,catalog,redemption}/` — the seven
  product-domain contracts. Each domain owns its messages and services.

## Adding a new schema

Add new `.proto` files under the owning `v1/<domain>/` directory. Then run
`make generate SCENARIO=token-economy` from `packages/proto/`, run the proto
lint target, and commit the regenerated artifacts in `packages/proto/gen/`.

## Why this layout

Keeping each product domain in its own directory makes contract ownership
match the API package boundary and permits scoped generation and review.
