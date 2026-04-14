# Resource Blueprints

Resource blueprints are the Phase 1 replacement for keeping speculative or stale resource implementations alive in `resources/`.

They preserve capability knowledge in a structured, searchable form without implying that a resource is currently implemented or supported by the active control plane.

When an old repo implementation should be removed while keeping the capability alive as a future candidate, pair the blueprint with the blueprint-only archival workflow described in [resource-blueprint-archival.md](resource-blueprint-archival.md).

## What a Blueprint Is

- A structured JSON record under `.vrooli/resources/blueprints/`
- A way to preserve future implementation knowledge
- A CLI-inspectable artifact through `vrooli resource blueprint ...`

## What a Blueprint Is Not

- Not an implemented resource
- Not evidence that `vrooli resource <name> ...` is supported
- Not a substitute for validation in a real scenario

## Storage and Schema

- Blueprint files live in `.vrooli/resources/blueprints/`
- Each file uses the schema at `.vrooli/schemas/resource-blueprint.schema.json`
- Filenames should match the canonical resource name, for example `terraform.json`

## Current CLI Surface

Phase 1 supports read and validation workflows:

```bash
vrooli resource blueprint list
vrooli resource blueprint info terraform
vrooli resource blueprint search network
vrooli resource blueprint validate
```

## Phase 1 Completion State

Phase 1 is now inventory-complete, not just seed-complete.

- Every resource classified as `blueprint` in `docs/resources/resource-phase0-inventory.md` now has a matching record under `.vrooli/resources/blueprints/`
- `go test ./internal/resources ./cmd/vrooli` now validates both:
  - blueprint schema and CLI behavior
  - drift between the Phase 0 inventory and the blueprint catalog
- The closeout validation bundle for Phase 1 is:
  - `vrooli resource blueprint validate`
  - `vrooli resource blueprint list`
  - `vrooli resource blueprint info terraform`
  - `vrooli resource blueprint search network`
  - `go test ./internal/resources ./cmd/vrooli`

That means the blueprint store is no longer a small illustrative batch. It is the materialized Phase 0 blueprint set.

Current catalog audit summary, as of `2026-04-11`:

- `53` blueprint records are present, matching the full Phase 0 `blueprint` set
- Template distribution:
  - `13` `docker-service`
  - `18` `manual-resource`
  - `8` `compose-service`
  - `6` `external-cli`
  - `5` `desktop-app`
  - `3` `cloud-api`
- Portability distribution:
  - `13` `full`
  - `34` `partial`
  - `6` `platform-specific`
- Every current record includes references, implementation notes, operational notes, and risks
- `replacement_for` currently mirrors the superseded legacy resource identity for the same canonical name. In this Phase 1 catalog, that often means a blueprint lists its own canonical resource name because it is preserving knowledge from the former `resources/<name>/` implementation rather than replacing a differently named resource

## Authoring Rules

- Use lowercase kebab-case for `name`
- Keep records honest about platform support and uncertainty
- Prefer concise implementation and operational notes over pseudo-code
- Use `replacement_for` when a blueprint captures knowledge from an older resource directory
- If the blueprint is preserving the old implementation for the same canonical resource identity, `replacement_for` may match `name`
- Keep `last_reviewed` current when materially editing a record

## Phase 1 Boundary

Phase 1 makes blueprints a supported concept, a validated data model, and the canonical structured home for the current Phase 0 blueprint-designated resources.

It does not yet:

- replace active implemented resources with manifest-driven drivers
- automatically archive an existing `resources/<name>/` implementation by itself

Phase 2 added deprecation/archive lifecycle support, and Phase 3 now allows blueprint-seeded template scaffolding through `vrooli resource template generate --from-blueprint <name>`.

Later cleanup work adds a distinct blueprint-only archival lifecycle for removing stale implementations from the repo without classifying the capability as deprecated.

Phase 3 also makes `suggested_template` enforceable rather than advisory-only: blueprint validation now checks that `integration_kind` and `suggested_template` obey the supported recommendation rules used by the template generator.

Those concerns belong to later phases in the migration plan.
