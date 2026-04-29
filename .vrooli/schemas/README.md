# Vrooli Schema Inventory

## Overview

`.vrooli/schemas/` contains a mix of:

- canonical authored schemas that define repo-owned manifest contracts
- generated schema artifacts derived from active resource manifests
- a small amount of older modular schema surface that should be treated as transitional until its live usage is confirmed

The most important source-of-truth rule is:

- `resources/<name>/resource.json` is the canonical source for resource manifest structure and resource-specific dependency authoring
- `.vrooli/schemas/resource-definitions.json` and `.vrooli/schemas/resource-catalog.json` are generated artifacts, not canonical authored configuration

## Current Inventory

### Canonical Authored Schemas

- `service.schema.json`
  Defines project-level and scenario-level `.vrooli/service.json` manifests.
  The current top-level properties include `service`, `dependencies`, `deployment`, `hostTools`, `hostSafeguards`, `lifecycle`, `ports`, `runtime`, and `version`.

- `operator-state.schema.json`
  Defines the per-install operator-state document at `.vrooli/operator-state.json`. Holds mutable operator choices (which scenarios/resources are enabled, per-scenario auto-restart overrides, host-tool and safeguard opt-ins) written by `vrooli-onboarding`. See [`docs/configuration/architecture.md`](../../docs/configuration/architecture.md) for the manifest-vs-state separation.

- `resource.schema.json`
  Defines `resources/<name>/resource.json` manifests for active resources and resource templates.

- `resources.schema.json`
  Defines the generic dependency wrapper shared by scenario/project resource declarations.
  This is the generic layer merged with resource-specific dependency schema during generated artifact creation.

- `common.schema.json`
  Shared definitions such as `semver`, `port`, `url`, and other reused primitives.

- `repo-contract.schema.json`
  Defines the schema for `.vrooli/repo-contract.json`.

- `package.schema.json`
  Defines the schema for package-governance manifests.

- `resource-blueprint.schema.json`
  Defines the schema for `.vrooli/resource-blueprints/*.json`.

- `deprecated-resources.schema.json`
  Defines the schema for `.vrooli/deprecated-resources.json`.

### Generated Artifacts

- `resource-definitions.json`
  Generated from active `resources/*/resource.json` manifests.
  Contains aggregated resource-specific dependency schemas plus the generated `resourceCatalog` object consumed by `service.schema.json`.

### Transitional / Review Candidates

- `deployment.schema.json`
- `lifecycle.schema.json`
- `scenarios.schema.json`

These files still exist in the directory, but they are not currently part of the most obvious live schema-consumer path. Keep them until their remaining usage is either confirmed or intentionally retired.

## Generated Artifact Pipeline

The generated artifacts are built by:

- `internal/resources/schema_artifacts.go`

The generation flow is:

1. Read every `resources/<name>/resource.json`
2. Extract each manifest's `dependency_schema`
3. Merge resource-specific dependency schema with `resources.schema.json#/definitions/resourceConfig`
4. Emit `resource-definitions.json`

Commands:

```bash
vrooli resource schema sync
vrooli resource schema validate
```

Automatic regeneration:

- `vrooli setup` regenerates these artifacts through the setup path

## Current Usage

### `service.schema.json`

`service.schema.json` is a live consumer of the generated artifacts:

- it references `resource-definitions.json#/resourceCatalog`
- that makes `resource-definitions.json` part of the current schema-validation chain for `service.json`

### `resource-definitions.json`

This file is still live today because:

- `service.schema.json` depends on it
- schema artifact validation checks it for drift
- setup regenerates it automatically

## Source Of Truth Rules

For resource-related schema work:

- author canonical resource shape in `resources/<name>/resource.json`
- author resource-specific dependency authoring shape in `resource.json.dependency_schema`
- do not hand-edit `resource-definitions.json`

For repo-aware structure work:

- update `.vrooli/repo-contract.json`
- update `repo-contract.schema.json`
- update repo-contract docs and tests in the same change

## Historical Notes

Older docs and planning material may still describe:

- `execution.schema.json`
- `serve.schema.json`
- a removed `build-aggregated-schemas.sh` script

Those references are historical. The current implementation uses:

- `service.schema.json` as the main authored service manifest schema
- `internal/resources/schema_artifacts.go` as the active generated-artifact implementation

## Validation

Useful validation entrypoints:

```bash
vrooli contract validate
vrooli resource schema validate
vrooli resource schema sync
make validate-repo-contract
```
