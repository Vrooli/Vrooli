# Storage Standardization Implementation Plan

**Status:** Proposed
**Scope:** Scenario runtime storage, resource runtime storage, repo-local legacy `data/` removal, and cross-platform/native-Go convergence
**Out of Scope:** Immediate code changes in this document; this is a planning artifact only

## 1. Goal

Standardize how Vrooli stores mutable runtime data so that:

- scenario source trees are not used as runtime data roots
- resource source trees are not used as runtime data roots
- the top-level repo `data/` folder becomes removable
- cross-platform/native-Go work has one clear storage target architecture
- new work stops reintroducing repo-local runtime state

This plan separates two domains that should align but not be conflated:

1. **Scenarios**
2. **Resources**

Scenarios should continue using the scenario runtime storage seam. Resources should get their own resource-specific storage/runtime seam.

---

## 2. Target Principles

- Repo source trees are immutable inputs, not runtime storage targets.
- Top-level `data/` is transitional debt, not target architecture.
- Scenarios own application behavior and app-runtime state.
- Resources own infrastructure capability and resource-runtime state.
- Shared storage/path logic should exist once per domain.
- `.vrooli/service.json` and `resource.json` remain declarative authority.
- Shell remains compatibility-only until retired.

---

## 3. Recommended Architecture

### 3.1 Scenarios

Scenarios should keep using `github.com/vrooli/api-core/storage` for mutable runtime filesystem state.

Scenario rules:

- all mutable filesystem writes go through `api-core/storage`
- structured persistence is declared in `scenarios/<name>/.vrooli/service.json`
- schema/init files live under `initialization/`
- scenario folders are deployable source trees, not runtime data roots

Scenario resolved storage should continue to look like:

- `<config-root>/vrooli/<scenario>/...`
- `<data-root>/vrooli/<scenario>/...`
- `<cache-root>/vrooli/<scenario>/...`
- `<logs-root>/vrooli/<scenario>/...`
- `<state-root>/vrooli/<scenario>/...`

### 3.2 Resources

Resources should **not** standardize on `api-core/storage`.

Resources need a resource-specific shared storage/path package because:

- many resources are not APIs
- many resources are Docker/compose/external-cli wrappers
- many resources need host/service storage, not scenario app storage
- resources need control-plane semantics more than app-runtime semantics

Recommended shared package target:

- `internal/resources/runtime/storage`

If it later needs to be consumed outside `internal/`, promote it intentionally. Do not start by putting it in `api-core`.

Resource resolved storage should look like:

- `<config-root>/vrooli/resources/<resource>/...`
- `<data-root>/vrooli/resources/<resource>/...`
- `<cache-root>/vrooli/resources/<resource>/...`
- `<logs-root>/vrooli/resources/<resource>/...`
- `<state-root>/vrooli/resources/<resource>/...`

This gives symmetry with scenarios without collapsing the domains together.

---

## 4. Canonical Storage Targets

### 4.1 Scenario Storage

Scenario runtime state belongs outside the repo and is resolved by `api-core/storage`.

Use storage classes intentionally:

- `config`: durable operator/user configuration
- `data`: primary mutable application data
- `cache`: rebuildable artifacts
- `logs`: diagnostics and operational logs
- `state`: checkpoints, locks, transient runtime state

### 4.2 Resource Storage

Resource runtime state belongs outside the repo and is resolved by the new resource storage layer.

Use the same class model where it helps:

- `config`
- `data`
- `cache`
- `logs`
- `state`

Examples:

- Docker service persisted volume data -> `data`
- generated configs/env files -> `config`
- snapshots and rebuildable indexes -> `cache` or `data` depending on semantics
- service logs -> `logs`
- pid files / sockets / lock files -> `state`

---

## 5. Migration Strategy

This work should be done as two related migrations:

1. scenario storage standardization
2. resource control-plane and storage standardization

They should use similar conventions but not the same implementation package by default.

### Phase 1: Define the standards

Write and land canonical docs for:

- scenario storage policy
- resource storage policy
- top-level `data/` as legacy/transitional
- canonical runtime roots and storage classes
- canonical env vars vs compatibility-only env vars

Recommended outputs:

- `docs/scenarios/storage.md`
- `docs/resources/storage.md`

Do not overfit scenario-private runtime details into the repo contract unless they are truly shared, stable, and version-worthy.

### Phase 2: Lock the resource target architecture

Define the target resource architecture before migrating implementations.

Specify:

- canonical resource folder layout
- what belongs in `cli/`
- what belongs in shared Go drivers/platform packages
- what belongs in per-resource code
- how retained shell code is labeled during migration

Resource template kinds that should be covered explicitly:

- `docker-service`
- `compose-service`
- `external-cli`
- `cloud-api`
- `desktop-app`
- `manual`

### Phase 3: Build shared resource runtime packages

Implement a resource-specific shared runtime layer that covers:

- storage/path resolution
- class roots
- path validation and safe joins
- ensure-dir helpers
- env rendering
- volume/source resolution
- health-check execution primitives
- log path discovery
- install/start/stop/status primitives where appropriate

This is the key step that prevents every resource from inventing path logic independently.

### Phase 4: Add enforcement before broad migration

Add checks so the repo stops regressing while migration is in progress.

Recommended checks:

- reject new mutable writes to `./data`
- reject new mutable writes to `${ROOT}/data`
- reject new mutable writes to `${APP_ROOT}/data`
- allow checked-in fixtures and explicit legacy exceptions only
- add manifest validation rules for resource volume sources
- add scenario checks for repo-local runtime writes
- add a clear legacy marker/exception mechanism for temporary holdouts

Without enforcement, migration will drift immediately.

### Phase 5: Migrate setup/lifecycle markers out of top-level `data/`

Move current setup/resource population markers out of root `data/` and into `.vrooli/` state.

Examples:

- `.setup-complete`
- `.resources-populated`
- resource population markers

Update:

- Go setup/lifecycle code
- retained shell compatibility code

This is a small, high-leverage migration that removes one of the last legitimate reasons for root `data/`.

### Phase 6: Migrate resource templates first

Update resource templates so all new resources are born correct.

Requirements:

- templates emit manifest-native storage patterns
- templates consume the shared resource storage/runtime packages
- template tests validate storage roots and portability assumptions
- new resources are not allowed to target repo-local `data/`

Do this before broad migration of old resources.

### Phase 7: Migrate active resources by archetype

Migrate active retained resources in batches.

Suggested order:

1. Docker-backed resources already close to target
   - `postgres`
   - `redis`
   - `qdrant`
   - `minio`
   - `vault`
2. Mixed manifest-native and shell-era storage/resources
   - `opencode`
   - `home-assistant`
   - `litellm`
3. Explicit repo-`data/` holdouts
   - `neo4j`
   - `sagemath`
4. External CLI / cloud API wrappers
5. Compose/manual edge cases

For each resource:

- move runtime paths out of the repo
- keep `resource.json` authoritative
- reduce shell to adapter/shim or remove it
- document migration notes if user data continuity matters

### Phase 8: Migrate scenarios with repo-local runtime data

Prioritize scenarios that still write to:

- `scenarios/<name>/data/...`
- `../data/...`
- repo-relative mutable storage roots

Likely early candidates:

- `app-issue-tracker`
- `browser-automation-studio`
- `vrooli-assistant`
- `prd-control-tower`

For each scenario:

- route file state through `api-core/storage`
- keep DB/init in `service.json` + `initialization/`
- add or update `docs/internal/STORAGE_AUDIT.md`
- remove assumptions that the repo tree is a writable runtime storage root

### Phase 9: Cut over env semantics

Scenarios:

- prefer `VROOLI_STORAGE_ROOT` and class-specific roots
- treat direct `VROOLI_DATA` use as compatibility-only where possible

Resources:

- define a canonical resource storage env surface
- either use a dedicated root such as `VROOLI_RESOURCE_STORAGE_ROOT`
- or keep env details encapsulated inside the resource storage package and drivers

Keep compatibility reads temporarily, but stop documenting legacy vars as target architecture.

### Phase 10: Remove top-level `data/` from the architecture

After migrations are complete:

- delete remaining product/runtime uses of top-level `data/`
- retain only real fixtures/testdata where appropriate
- remove compatibility code that assumes repo-local runtime roots
- keep `**/data/**` excluded in bundle profiles
- treat new top-level runtime `data/` uses as regressions

---

## 6. Recommended Resource Organization

Target resource structure:

```text
resources/
  postgres/
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

Rules:

- `resource.json` is the declarative contract and authority
- `cli/` is the executable entrypoint
- `internal/` contains resource-specific Go behavior
- shared driver/runtime logic should not be copied per resource
- `lib/*.sh` and `config/defaults.sh` may remain temporarily during migration, but are not target architecture

### What should live in shared resource platform code

- path/storage resolution
- common driver behavior
- health-check plumbing
- environment rendering/exports
- shared log/status handling
- portability classification and validation helpers

### What should live in per-resource code

- truly resource-specific install/start/status behavior
- resource-specific config translation
- resource-specific health probes when common ones are insufficient

---

## 7. Ideal Scenario End-State

Example of an ideal scenario that fully adopts the target architecture:

```text
scenarios/
  example-scenario/
    .vrooli/
      service.json
      testing.json
      deployment/
        deployment-report.json

    README.md
    PRD.md
    Makefile

    docs/
      README.md
      internal/
        STORAGE_AUDIT.md
        PORTABILITY_AUDIT.md
        SEAMS.md

    initialization/
      storage/
        postgres/
          schema.sql
          seed.sql
      configuration/
        app-config.json

    api/
      go.mod
      go.sum
      main.go
      internal/
        app/
        handlers/
        domain/
        repository/
        service/
        runtime/
      pkg/

    cli/
      main.go

    ui/
      package.json
      src/
      public/

    requirements/
      index.json

    test/
      smoke/
      integration/
      fixtures/
```

### Intentionally absent

- no scenario-owned runtime `data/` directory
- no mutable runtime logs/state/cache under the scenario tree
- no SQLite DB under `scenarios/example-scenario/data/...`
- no repo-local uploads/session files/temp files

### Runtime state location

- config/state/cache/logs/data resolved by `api-core/storage`
- DB schemas declared in `.vrooli/service.json`
- initialization artifacts under `initialization/`
- durable database content stored in configured resources, not repo-local folders

### Runtime behavior model

- `service.json` declares dependencies and initialization
- API constructs `api-core/storage` resolver once at startup
- filesystem writes go to classed external paths
- database connections come from injected resource env
- scenario tree remains replaceable and bundle-safe

---

## 8. Decision Summary

Recommended final direction:

- scenarios continue using `api-core/storage`
- resources get a new resource-specific storage/runtime package
- top-level `data/` is treated as migration debt
- resource templates and validation are updated before broad migration
- active resources and scenarios are then migrated in batches with enforcement enabled

This preserves the clear scenario runtime seam already in place while giving resources a control-plane-native storage model instead of forcing them into an API-specific package.
