# Docker-Service Native Proposal

## Purpose

Define the proper future-state contract for `docker-service` resources in Vrooli.

This proposal is based on the live repo state, not just the migration plan. It focuses on:

- what the `docker-service` archetype should mean
- which shell-era behaviors are still live and must be replaced natively
- which shell-era files are baggage and should not be carried forward
- how to migrate the current active `docker-service` resources without letting compatibility code become the contract

## Executive Summary

The correct target for `docker-service` is:

```text
canonical template = zero Bash
manifest = authoritative
Go driver = authoritative for standard lifecycle
native env export model = authoritative for scenario dependency injection
compatibility code = explicit, isolated, temporary, removable
```

The current repo is not there yet.

What already exists:

- a working shared Go `docker-service` driver for standard lifecycle/status/logs
- manifest-native discovery via `resources/<name>/resource.json`
- partial native resource environment loading via `internal/resources/metadata.go`

What is still shell-era and live:

- `config/exports.sh` for some resources
- direct scenario sourcing of `exports.sh`
- shell-based database bootstrap for Postgres
- many resource-local custom commands living in `cli.sh` + `lib/*.sh`
- shell-era config/test/docs assumptions around `defaults.sh`, `runtime.json`, `messages.sh`, and `capabilities.yaml`

The migration goal is not "delete Bash blindly." It is to make Bash non-authoritative and then removable.

## Current Active Docker-Service Set

Active resources with `driver = "docker-service"`:

- `browserless`
- `comfyui`
- `litellm`
- `minio`
- `neo4j`
- `ollama`
- `postgres`
- `qdrant`
- `questdb`
- `redis`
- `sagemath`
- `searxng`
- `unstructured-io`
- `vault`

Reference:

- [drivers.go](/home/matthalloran8/Vrooli/internal/resources/drivers.go)
- `go run ./cmd/vrooli resource list --json`

## What Is Already Native

The shared Go `docker-service` driver already owns:

- `install`
- `start`
- `restart`
- `stop`
- `uninstall`
- `status`
- `logs`
- health-check execution for manifest-declared checks

The driver currently uses:

- `resource.json`
- Docker image/container inspection
- manifest runtime env/ports/volumes
- typed platform gating

References:

- [drivers.go](/home/matthalloran8/Vrooli/internal/resources/drivers.go:243)
- [manifest.go](/home/matthalloran8/Vrooli/internal/resources/manifest/manifest.go)

This is the correct baseline and should become the only standard lifecycle path for `docker-service`.

## Live Shell-Era Dependencies That Still Matter

### 1. Environment Injection

The old shell system exported scenario-facing env vars through `config/exports.sh`, sometimes falling back to `defaults.sh`.

This behavior is still live in at least four resources:

- `ollama`
- `postgres`
- `qdrant`
- `redis`

References:

- [resources/ollama/config/exports.sh](/home/matthalloran8/Vrooli/resources/ollama/config/exports.sh)
- [resources/postgres/config/exports.sh](/home/matthalloran8/Vrooli/resources/postgres/config/exports.sh)
- [resources/qdrant/config/exports.sh](/home/matthalloran8/Vrooli/resources/qdrant/config/exports.sh)
- [resources/redis/config/exports.sh](/home/matthalloran8/Vrooli/resources/redis/config/exports.sh)

There is already a partial native replacement:

- [metadata.go](/home/matthalloran8/Vrooli/internal/resources/metadata.go)
- [ports.go](/home/matthalloran8/Vrooli/internal/ports/ports.go:600)

The native path currently derives env from:

- `scripts/resources/port_registry.json`
- `.vrooli/schemas/resource-definitions.json`
- secrets store
- hard-coded special cases

This is real and in use, but incomplete relative to the old shell export surface.

### 2. Direct Scenario Sourcing Of Resource Shell Files

At least some scenario scripts still source resource shell exports directly:

- [run-migrations.sh](/home/matthalloran8/Vrooli/scenarios/tech-tree-designer/scripts/run-migrations.sh:8)
- [verify-postgres-data.sh](/home/matthalloran8/Vrooli/scenarios/tech-tree-designer/scripts/verify-postgres-data.sh:9)

This means the native env contract must become explicit before those shell files can disappear.

### 3. Postgres Database Bootstrap

Scenario lifecycle still shells into Postgres resource libraries for:

- database creation
- schema application
- migration application

References:

- [internal/lifecycle/setup.go](/home/matthalloran8/Vrooli/internal/lifecycle/setup.go:63)
- [scripts/lib/utils/lifecycle.sh](/home/matthalloran8/Vrooli/scripts/lib/utils/lifecycle.sh:168)
- [resources/postgres/lib/common.sh](/home/matthalloran8/Vrooli/resources/postgres/lib/common.sh)
- [resources/postgres/lib/database.sh](/home/matthalloran8/Vrooli/resources/postgres/lib/database.sh)

This is one of the most important remaining native gaps.

## Shell-Era File Classes

These file classes still appear across active `docker-service` resources:

- `config/defaults.sh`
- `config/runtime.json`
- `config/schema.json`
- `config/messages.sh`
- `config/capabilities.yaml`
- `config/exports.sh`
- `config/agents.conf`
- `cli.sh`
- `lib/*.sh`

These should not all survive. They need classification.

## File Classification Rules

Every shell-era file should be classified into exactly one of these buckets:

### `still-authoritative`

Live behavior depends on it today and that behavior must migrate.

Examples:

- `config/exports.sh` for `postgres`, `redis`, `qdrant`, `ollama`
- Postgres DB bootstrap logic in `lib/database.sh`

### `replace-with-native-structure`

Behavior is still needed, but the file format or location should change.

Examples:

- `config/runtime.json`
- `config/schema.json`
- `config/secrets.yaml`
- `config/agents.conf`

### `compatibility-only`

Temporary bridge for custom commands that are still live but not yet migrated.

Examples:

- `cli.sh` + `lib/*.sh` for rich custom command surfaces such as `minio`, `neo4j`, `sagemath`, `searxng`

### `delete`

Historical baggage with no justified future role.

Likely candidates:

- `messages.sh`
- `capabilities.yaml`
- shell-only test fixtures and old universal-contract assumptions

## Proposed Native Docker-Service Contract

## 1. Canonical Resource Layout

```text
resources/<name>/
├── README.md
├── resource.json
├── docs/
│   └── OPERATIONS.md
├── config/
│   ├── defaults.json
│   ├── schema.json
│   └── env-exports.json
├── docker/
│   └── optional static config assets
└── test/
    ├── smoke.json
    └── integration.json
```

Rules:

- no Bash files in the canonical template
- no `cli.sh` in the canonical template
- no `lib/*.sh` in the canonical template
- no shell-era config names in the canonical template

## 2. Standard Lifecycle Ownership

For `driver = docker-service`, the following must be Go-owned:

- install
- start
- restart
- stop
- uninstall
- status
- logs
- health

If any of those still require shell, the resource is not fully native.

## 3. Native Scenario Environment Contract

The shell-era `exports.sh` behavior should be replaced with an explicit native model.

Suggested addition:

```json
"environment_exports": {
  "static": {
    "POSTGRES_HOST": "localhost",
    "POSTGRES_SSLMODE": "disable"
  },
  "derived": [
    {
      "name": "POSTGRES_URL",
      "from": "postgres_connection_url"
    },
    {
      "name": "DATABASE_URL",
      "from": "postgres_connection_url"
    }
  ]
}
```

The exact schema can vary, but the important point is:

- exported scenario-facing env vars must be declared and generated natively
- resource env injection must not depend on shell sourcing
- derived URLs and credentials must have typed native construction logic

This should replace the current hidden dependence on:

- `exports.sh`
- `.vrooli/schemas/resource-definitions.json`
- hard-coded special cases in Go

## 4. Compatibility Layout

If a resource still has custom shell behavior during migration, it should be isolated:

```text
resources/<name>/
├── resource.json
├── compat/
│   ├── README.md
│   ├── bridge.json
│   └── shell/
│       ├── cli.sh
│       ├── config/
│       └── lib/
```

Rules:

- compatibility code is not part of the canonical template
- compatibility code is not authoritative for standard lifecycle
- compatibility code must have explicit ownership and removal criteria

## Audit Of Active Docker-Service Resources

## Low Custom Burden

Good first targets for complete native cleanup:

- `browserless`
- `litellm`
- `redis`
- `questdb`
- `vault`

These have relatively small custom command surfaces or already narrow retained compatibility behavior.

## Medium Custom Burden

Likely second wave:

- `comfyui`
- `ollama`
- `postgres`
- `qdrant`
- `unstructured-io`

These still have important custom operations and some live env/export or content workflows.

## High Custom Burden

Likely last wave:

- `minio`
- `neo4j`
- `searxng`
- `sagemath`

These have large shell command surfaces and should not be migrated by mechanically porting every old subcommand.

## Recommended Cohort Plan

### Cohort A: Contract And Runtime Foundations

Do first:

1. Define native env export contract for `docker-service`
2. Replace shell-based scenario env loading with Go-native loading only
3. Replace Postgres DB bootstrap with native Go logic
4. Define compatibility isolation layout and contract checks

Without this, resource-by-resource migration will keep leaking shell assumptions back into the platform.

### Cohort B: Gold Standard Resources

Use these as the first clean targets:

- `redis`
- `postgres`
- `browserless`

Desired result:

- no standard lifecycle dependence on shell
- no scenario dependence on `exports.sh`
- explicit native env export support
- compatibility only for justified custom commands

### Cohort C: Finish The Low/Medium Set

- `litellm`
- `questdb`
- `vault`
- `ollama`
- `qdrant`
- `comfyui`
- `unstructured-io`

### Cohort D: Re-design Instead Of Blind Porting

For these resources, first decide which custom commands are actually worth preserving:

- `minio`
- `neo4j`
- `searxng`
- `sagemath`

Do not port the full shell CLI surface by default.

## Repo Contract And Enforcement Changes

The repo contract already says the future-state platform is Go-native:

- [repo-contract.json](/home/matthalloran8/Vrooli/.vrooli/repo-contract.json)
- [repo-contract.md](/home/matthalloran8/Vrooli/docs/repo-contract.md)

The resource contract should now be tightened to match that claim.

Recommended enforcement:

### Resource Template Rules

- canonical `docker-service` template generates no Bash
- canonical `docker-service` template includes native env export declaration
- canonical `docker-service` template includes no shell-first docs

### Active Resource Rules

For `driver = docker-service`:

- standard lifecycle commands must be native
- compatibility code must live under explicit compatibility layout if present
- `cli.sh` at resource root should not remain a silent default forever
- `exports.sh` must not be required for scenario env injection

### Documentation Rules

Mark these as historical/transitional:

- shell universal contract docs
- `defaults.sh`/`messages.sh` requirements
- `runtime.json` as an operator-facing source of truth

## Critical Unknowns To Resolve

These need explicit follow-up investigation or design:

1. How `.vrooli/schemas/resource-definitions.json` is authored and whether it should survive at all
2. Exact native schema for resource-provided scenario env vars
3. Whether `config/schema.json` should remain resource-local or be replaced by manifest-native definitions
4. Which custom commands are actually used by repo code vs only documented in old PRDs/READMEs
5. Whether compatibility should remain shell-based temporarily or move to Go wrappers around retained behaviors

## Recommended Immediate Next Steps

1. Add a first-class native env export section to `resource.json`
2. Replace `internal/ports` dependency on `resource-definitions.json` + special cases with manifest-native env export resolution
3. Replace Postgres shell bootstrap with native Go DB bootstrap helpers
4. Add contract checks that canonical resource templates contain no Bash files
5. Add contract checks that `docker-service` resources cannot rely on root-level `cli.sh` for standard lifecycle
6. Move any retained shell compatibility under explicit compatibility paths
7. Audit repo usage of custom resource commands before migrating large custom command surfaces

## Bottom Line

The `docker-service` archetype is close to viable as a real professional standard, but not finished.

The real blockers are not the shared driver. They are:

- incomplete native env export parity
- shell-based side effects in lifecycle/bootstrap
- unbounded custom shell command surfaces
- lack of explicit compatibility isolation

Fix those four things, and `docker-service` becomes a real cross-platform native archetype instead of a manifest layered on top of hidden Bash.
