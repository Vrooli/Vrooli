# Docker-Service Native Proposal

## Purpose

Define the proper future-state contract for `docker-service` resources in Vrooli.

This proposal is based on the live repo state, not just the migration plan. It focuses on:

- what the `docker-service` archetype should mean
- which shell-era behaviors are still live and must be replaced natively
- which shell-era files are baggage and should not be carried forward
- how to migrate the current active `docker-service` resources without letting compatibility code become the contract

## Executive Summary

Status update:

- the native env export contract is now implemented
- `config/runtime.json` and `config/schema.json` have been removed from active resources
- `config/exports.sh` has been removed from the migrated docker-service resources that used to require it
- `.vrooli/schemas/resource-definitions.json` is now generated from `resource.json`
- remaining shell surfaces are compatibility code, not runtime authority

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
- native scenario start/restart env assembly through `internal/ports.BuildEnvironment()`, which already bypasses `config/exports.sh`

What is still shell-era and live:

- shell-based database bootstrap for Postgres
- many resource-local custom commands living in `cli.sh` + `lib/*.sh`
- shell-era config/test/docs assumptions around `defaults.sh`, `runtime.json`, `messages.sh`, and `capabilities.yaml`

The migration goal is not "delete Bash blindly." It is to make Bash non-authoritative and then removable.

This is therefore replacement work, not greenfield addition:

- standard lifecycle is already mostly replaced for `docker-service`
- scenario start/restart env assembly is already mostly replaced
- what remains is to close the parity gaps, make the contract explicit, add validation, and then delete the leftover shell path

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

Current scenario-facing env contract status:

- explicitly native: `browserless`, `comfyui`, `minio`, `ollama`, `postgres`, `qdrant`, `questdb`, `redis`, `searxng`, `unstructured-io`, `vault`
- intentionally not yet given scenario env exports: `litellm`, `neo4j`, `sagemath`

The remaining set is intentional. A repo audit did not find active scenario dependency consumers for `litellm`, `neo4j`, or `sagemath`, so this proposal does not invent scenario-facing env contracts for them prematurely.

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

Scenario lifecycle environment assembly is also already native on the main start/restart path:

- [internal/lifecycle/lifecycle.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle.go:168)
- [internal/ports/ports.go](/home/matthalloran8/Vrooli/internal/ports/ports.go:352)

Current start path:

```text
Runner.Start()
  -> Ports.BuildEnvironment()
  -> loadResourceEnvironment()
  -> resources.LoadResourceEnvironment()
  -> inject env into setup/develop/test phases
```

Important implication:

- `vrooli scenario start` and `restart` already ignore `config/exports.sh`
- this is not inherently broken, because a native replacement already exists
- but native parity with the old shell export surface is incomplete, so there is still drift risk

## Live Shell-Era Dependencies That Still Matter

### 1. Environment Injection

The old shell system exported scenario-facing env vars through `config/exports.sh`, sometimes falling back to `defaults.sh`.

This behavior is still live in at least four resources:

- `ollama`
- `postgres`
- `qdrant`
- `redis`

Those shell export files were part of the migration input and have now been removed in favor of manifest-native `environment_exports`.

There is already a partial native replacement:

- [metadata.go](/home/matthalloran8/Vrooli/internal/resources/metadata.go)
- [ports.go](/home/matthalloran8/Vrooli/internal/ports/ports.go:600)

The native path previously derived env from:

- `scripts/resources/port_registry.json`
- `.vrooli/schemas/resource-definitions.json`
- secrets store
- hard-coded special cases

That path was real and in use, but incomplete relative to the old shell export surface. The implemented target is now explicit `resource.json.environment_exports`, with legacy fallback removed.

Former parity gaps that required native replacement:

- `POSTGRES_URL` and `DATABASE_URL` are handled natively through `applyPostgresOverride`, but this logic is special-cased in `internal/ports`
- `REDIS_URL` is still a shell-era concept not reproduced natively in the current generic path
- `QDRANT_URL` and `QDRANT_GRPC_URL` are still shell-era concepts not reproduced natively
- `OLLAMA_URL` is still a shell-era concept not reproduced natively

So the correct question is not "should we add native env export logic?" The repo already has some.
The correct question is "how do we replace the incomplete native-plus-shell hybrid with one explicit native contract?"

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

They also need explicit removal planning so contributors are not left guessing which files still matter.

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

This should be modeled directly in `resource.json`, not in a sidecar shell file and not in `.vrooli/schemas/resource-definitions.json`.

Suggested addition:

```json
"environment_exports": {
  "static": {
    "POSTGRES_HOST": "localhost",
    "POSTGRES_SSLMODE": "disable"
  },
  "from_ports": {
    "POSTGRES_PORT": "postgresql"
  },
  "from_runtime_env": [
    "POSTGRES_USER",
    "POSTGRES_PASSWORD",
    "POSTGRES_DB"
  ],
  "derived": {
    "POSTGRES_URL": {
      "template": "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSLMODE}"
    },
    "DATABASE_URL": {
      "template": "${POSTGRES_URL}"
    }
  }
}
```

This proposal intentionally keeps the structure small and readable:

- `static`
  - literal values the resource always exports
- `from_ports`
  - env var name -> manifest port name
- `from_runtime_env`
  - selected keys copied from `runtime.env` after secrets/default resolution
- `derived`
  - computed values built from templates

This makes the contract inspectable in one place and avoids hiding behavior in Bash, generated JSON, or hard-coded Go maps.

The important point is:

- exported scenario-facing env vars must be declared and generated natively
- resource env injection must not depend on shell sourcing
- derived URLs and credentials must have typed native construction logic

This should replace the current hidden dependence on:

- `exports.sh`
- `.vrooli/schemas/resource-definitions.json`
- hard-coded special cases in Go

### 3.1 Example Flow

Target flow:

```text
scenario .vrooli/service.json
  -> declares dependency on postgres
  -> may supply overrides such as database/schema/model/base_url
        |
        v
resource.json
  -> runtime
  -> ports
  -> environment_exports
        |
        v
native resource env resolver
  -> resolves secrets/defaults
  -> maps named ports
  -> applies scenario overrides
  -> renders derived values
  -> validates collisions/ambiguity
        |
        v
final scenario env
  -> setup/develop/test/runtime phases
```

Example scenario override:

```json
"dependencies": {
  "resources": {
    "postgres": {
      "enabled": true,
      "required": true,
      "database": "tech_tree_designer"
    }
  }
}
```

Result:

```text
resource default contract
  -> exports POSTGRES_DB and derived URLs

scenario dependency override
  -> replaces POSTGRES_DB with tech_tree_designer

resolver recomputes
  -> POSTGRES_URL
  -> DATABASE_URL
```

### 3.2 Responsibility split

The future-state responsibility split should be:

```text
internal/ports
  -> scenario runtime port allocation only

internal/resources/env (or similar)
  -> resource-provided scenario env resolution
  -> environment_exports evaluation
  -> dependency override application
  -> derived URL generation
  -> collision/conflict detection
```

`internal/ports` should not remain the long-term home for resource env semantics.

Keep in `internal/ports`:

- scenario port range allocation
- lock management
- runtime port reuse
- listener/lock collision handling

Move out of `internal/ports`:

- resource env export semantics
- resource-specific overrides
- derived resource URL construction

Suggested package shape:

```text
internal/resources/env/
├── resolver.go
├── overrides.go
├── templates.go
├── validate.go
└── types.go
```

Responsibilities:

- `resolver.go`
  - build final exported env map for one resource dependency
- `overrides.go`
  - apply scenario dependency overrides such as `database`, `schema`, `models`
- `templates.go`
  - render derived env values safely and deterministically
- `validate.go`
  - local and scenario-level collision checks
- `types.go`
  - typed structures for env export declarations and diagnostics

Migration note:

- `internal/resources/metadata.go` should become a short-term adapter or be retired entirely once the native resolver owns the contract
- `.vrooli/schemas/resource-definitions.json` should no longer be used as an authoritative source for resource export semantics
- `internal/ports` should call the new resolver rather than continuing to own resource env behavior itself

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
- compatibility code must not be required for `scenario start` or `scenario restart`

Suggested `compat/bridge.json` fields:

```json
{
  "owner": "resources-team",
  "reason": "custom admin commands not yet ported",
  "authoritative_for": [],
  "commands": ["backup", "restore"],
  "decision_deadline": "2026-06-30",
  "removal_criteria": [
    "native env resolver shipped",
    "native validation shipped",
    "all scenario callers migrated"
  ]
}
```

## Validation And Conflict Detection

The native contract should not just replace shell behavior. It should be stricter and easier to audit.

### Resource-local validation

For each `docker-service` resource:

- detect duplicate port names
- detect duplicate fixed host ports within the same resource
- detect duplicate exported environment variable names
- detect invalid derived env templates
- detect references to missing ports, runtime values, or secrets

### Scenario-level validation

For one scenario with multiple enabled resources:

- detect overlapping exported env keys across enabled resources
- detect resource exports that collide with scenario-defined env vars
- detect invalid scenario overrides that break derived env output

### Cross-resource static validation

Across active `docker-service` resources:

- detect conflicting fixed host port declarations
- detect exported env names that are claimed by multiple resources incompatibly

### Recommended command surface

Expose this through native CLI validation:

```text
vrooli resource validate
vrooli resource validate <name>
vrooli scenario validate-env <scenario>
```

Recommended behavior:

- `vrooli resource validate`
  - validates manifest shape, env export declarations, local collisions, and fixed-port conflicts across resources
- `vrooli resource validate <name>`
  - validates one resource and prints actionable diagnostics
- `vrooli scenario validate-env <scenario>`
  - resolves enabled dependency exports
  - prints the final env set
  - reports collisions, overrides, and ambiguity

Recommended output shape:

```text
$ vrooli resource validate postgres
resource: postgres
status: invalid

errors:
- duplicate exported env key: DATABASE_URL
- derived export POSTGRES_URL references unknown variable: POSTGRES_SSLMODE

warnings:
- fixed host port 5433 overlaps with resource questdb
```

```text
$ vrooli scenario validate-env tech-tree-designer
scenario: tech-tree-designer
status: invalid

resource exports:
- postgres -> POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_URL, DATABASE_URL
- redis -> REDIS_HOST, REDIS_PORT, REDIS_URL

overrides:
- postgres.database = tech_tree_designer

errors:
- env key collision: DATABASE_URL provided by postgres and scenario api.env
```

Recommended command ownership:

- `vrooli resource validate`
  - resource manifest schema validation
  - env export declaration validation
  - cross-resource fixed-port conflict detection
- `vrooli scenario validate-env`
  - dependency resolution preview
  - final merged env inspection
  - scenario-level env collision detection

These commands should exist before the docker-service template is updated.

## Cleanup And Removal Plan

The migration should explicitly name what gets added, what gets replaced, and what gets deleted.

### Add

- `resource.json.environment_exports`
- native resource env resolver package
- native validation commands and diagnostics
- explicit `compat/bridge.json` when temporary shell compatibility is still needed

### Replace

- `internal/resources/metadata.go` env export semantics
- `internal/ports` resource env assembly behavior
- Postgres shell bootstrap in scenario setup
- direct scenario sourcing of `resources/*/config/exports.sh`

### Delete After Validation

- `config/exports.sh`
- shell fallback env loading from `scripts/lib/network/ports.sh`
- `.vrooli/schemas/resource-definitions.json` as an authoritative env-export source
- any `cli.sh` or `lib/*.sh` paths still serving standard lifecycle or scenario env injection

Deletion gates:

1. Native parity tests exist for every exported variable still relied on by scenarios.
2. `vrooli resource validate` passes for all active docker-service resources.
3. `vrooli scenario validate-env` passes for scenarios using those resources.
4. All direct scenario sourcing of resource shell exports has been removed.
5. Native Postgres bootstrap is in place.

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
2. Replace the current partial `internal/resources/metadata.go` + `internal/ports` hybrid with an explicit native env resolver
3. Replace shell-based scenario env loading and side-channel `exports.sh` sourcing with native loading only
4. Replace Postgres DB bootstrap with native Go logic
5. Define compatibility isolation layout and contract checks
6. Add resource-local, scenario-level, and cross-resource conflict validation
7. Add the native CLI validation surface

Do not update the canonical `docker-service` template until Cohort A is implemented and validated.

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
- template changes happen only after the native env/export/validation foundation is landed

### Active Resource Rules

For `driver = docker-service`:

- standard lifecycle commands must be native
- compatibility code must live under explicit compatibility layout if present
- `cli.sh` at resource root should not remain a silent default forever
- `exports.sh` must not be required for scenario env injection
- shell-era direct scenario sourcing must be treated as migration debt and removed
- shell-era side-channel files must have explicit `migrate`, `compat`, or `delete` disposition

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
2. Introduce a dedicated native resource env resolution package and move env semantics out of `internal/ports`
3. Replace `internal/ports` dependency on `resource-definitions.json` + special cases with manifest-native env export resolution
4. Replace Postgres shell bootstrap with native Go DB bootstrap helpers
5. Add resource-local and scenario-level conflict validation
6. Expose validation through native `vrooli` commands
7. Add contract checks that canonical resource templates contain no Bash files
8. Add contract checks that `docker-service` resources cannot rely on root-level `cli.sh` for standard lifecycle
9. Move any retained shell compatibility under explicit compatibility paths
10. Audit repo usage of custom resource commands before migrating large custom command surfaces
11. Only after steps 1-10 are landed and validated, update the canonical `docker-service` template

## Bottom Line

The `docker-service` archetype is close to viable as a real professional standard, but not finished.

The real blockers are not the shared driver. They are:

- incomplete native env export parity
- shell-based side effects in lifecycle/bootstrap
- unbounded custom shell command surfaces
- lack of explicit compatibility isolation

The sequencing matters:

```text
first:
  native env/export contract
  native validation
  native bootstrap replacement
  explicit cleanup/removal plan

then:
  update docker-service template
  migrate resources against the finalized contract
```

Fix those four things, and `docker-service` becomes a real cross-platform native archetype instead of a manifest layered on top of hidden Bash.
