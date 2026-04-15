# Resource Storage

This page defines the target storage policy for resources at the platform level.

It exists because resource manifests and control-plane docs already define metadata and lifecycle expectations, but resource data organization has not yet been documented clearly enough as one canonical standard.

## Current Rule

Resources must not treat the repo source tree as their long-term runtime storage root.

In particular, new resource work should not introduce or reinforce:

- `${ROOT}/data/...`
- `${APP_ROOT}/data/...`
- top-level repo `data/` as the normal persistent storage root for resource runtime state

Repo-local resource data remains part of the transitional reality in some retained resources, but it is not the target architecture.

When a retained resource still has to use repo-local runtime paths temporarily, its manifest must opt in explicitly with:

- `legacy_repo_data_allowed: true`

That flag is a migration marker, not a design endorsement.

## Target Direction

Resources should converge on a resource-specific shared storage/runtime layer owned by the resource control plane.

Resources should **not** standardize on `api-core/storage` as their final storage abstraction.

Reason:

- many resources are not APIs
- many resources are Docker/compose/external-cli wrappers
- many resources need host/service storage rather than scenario app storage
- resources are primarily a control-plane domain, not a scenario API domain

Recommended shared implementation target:

- `internal/resources/runtime/storage`

If this later needs to move to a promoted shared package, that should be an intentional follow-up rather than an assumption.

## Canonical Storage Classes

Resources should use the same storage-class vocabulary where useful:

- `config`
  - generated config files, `.env` files, operator-managed resource config
- `data`
  - primary persistent service data and mounted durable volumes
- `cache`
  - rebuildable snapshots, temp indexes, import/export scratch artifacts when safe to recreate
- `logs`
  - service logs and diagnostics
- `state`
  - pid files, sockets, locks, transient control-plane runtime state

Target resolved paths should look like:

- `<config-root>/vrooli/resources/<resource>/...`
- `<data-root>/vrooli/resources/<resource>/...`
- `<cache-root>/vrooli/resources/<resource>/...`
- `<logs-root>/vrooli/resources/<resource>/...`
- `<state-root>/vrooli/resources/<resource>/...`

## Manifest Authority

For resources, the canonical declarative authority remains:

- `resources/<name>/resource.json`

Storage-related resource behavior should be described there through:

- runtime volumes
- environment exports
- driver/runtime metadata
- health and orchestration settings

But manifests should point at canonical resource runtime roots, not repo-local `data/` paths.

## Resource Folder Organization

Target resource structure:

```text
resources/
  <name>/
    resource.json
    README.md
    docs/
    initialization/
    cli/
      main.go
    internal/
      install/
      runtime/
      status/
      health/
      env/
```

Guidance:

- `resource.json` is the declarative contract
- `cli/` is the executable entrypoint
- per-resource Go code lives under resource-local Go code, not shell libraries, in the target architecture
- shared logic should live in common resource control-plane packages rather than copied across resources
- shell files may remain as transitional adapters while migration is underway, but they are not the design center

## Shell Compatibility

Retained shell-era paths and defaults are transitional compatibility behavior.

They should be treated as:

- migration debt
- compatibility shims
- non-authoritative implementation detail

They should not be copied into new resources or new templates.

## Top-Level `data/`

From the resource perspective, the top-level repo `data/` folder is also legacy/transitional.

Today it still contains active runtime data for some resources, but that is not the target architecture.

The end goal is:

- resource source in `resources/<name>/...`
- resource runtime state outside the repo
- manifests and shared control-plane code as the canonical authority

## Relationship To Scenario Storage

Resources and scenarios should align philosophically but not share the same implementation package by default.

Recommended split:

- scenarios use `api-core/storage`
- resources use a resource-specific control-plane storage layer

This keeps scenario app-runtime concerns separate from resource service/control-plane concerns.

## Transitional Reality

The active tree currently contains a mix of patterns:

- some resources already use home-scoped or `${VROOLI_DATA}`-style paths
- some resources still point at repo-local `data/`
- some resources remain split between manifest-native Go control-plane behavior and shell-era runtime path conventions

That mixed state should be treated as migration input, not as the standard.

## Migration Rule

For future resource work:

- do not introduce new repo-local runtime storage roots
- prefer manifest-native path semantics
- prefer shared Go control-plane path/storage logic
- keep shell-only path logic isolated and clearly transitional when it still exists
- require explicit `legacy_repo_data_allowed: true` on retained manifests that have not migrated yet
