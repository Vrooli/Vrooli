# Repo Contract

## Purpose

The repository contract defines the future-state Vrooli repository structure that repo-aware tooling is allowed to depend on. It prevents shared packages, internal platform code, and repo-aware scenarios from hard-coding their own assumptions about:

- repo root detection
- canonical top-level directories
- canonical scenario and resource layout
- root-relative glob semantics
- structural environment variable names
- repo-aware bundle/include profiles

The authoritative artifacts are:

- [.vrooli/repo-contract.json](../.vrooli/repo-contract.json)
- [.vrooli/schemas/repo-contract.schema.json](../.vrooli/schemas/repo-contract.schema.json)

## Implementation Status

Phase 1 implementation status:

- the versioned contract is landed
- the schema is landed
- repo conformance and drift tests are landed
- validation entrypoints are landed

Landed:

- Phase 1 contract definition and validation
- Phase 2 Go adapter implementation in `packages/repo-contract-go`
- the initial Phase 3 shared-package integration slice in:
  - `packages/api-core/scenario`
  - `packages/cli-core/cliutil/sandbox`
  - `internal/scenario`

Still deferred to later phases:

- broader drift checks for remaining direct consumers outside the landed migration set

Phase 1 includes:

- a versioned JSON contract
- a JSON schema for that contract
- versioning and compatibility rules
- normalization rules for slash-style repo-relative paths
- conformance tests against the live repo
- validation entrypoints suitable for CI

Phase 1 excluded broad consumer migration and runtime behavior beyond validation/documentation. Those exclusions no longer apply to the landed Phase 2 adapter and the current shared-package Phase 3 slice.

## Phase 1 Completion Rules

Phase 1 should be considered complete only when all of the following remain true:

- `.vrooli/repo-contract.json` stays aligned with the future-state repo shape
- `.vrooli/schemas/repo-contract.schema.json` enforces the current contract shape
- `vrooli contract validate` remains the canonical low-level contract validation entrypoint
- `make hygiene` remains the CI/automation wrapper for precommit readiness
- `internal/repocontract` catches schema drift, semantic drift, and legacy-path regressions
- remaining non-migrated consumers are clearly documented as migration targets rather than contract authority

## Shared Package Adoption

The current shared-package baseline is:

- `packages/repo-contract-go` is the authoritative Go adapter for contract-backed repo/layout semantics
- `packages/api-core/scenario` uses the contract for repo-root detection, scenario-root discovery, and canonical manifest lookup
- `packages/cli-core/cliutil/sandbox` uses the contract for repo-root defaults, sandbox scope matching, and scenario path resolution
- `internal/scenario` uses the contract-backed adapter for canonical scenario layout and sandbox path resolution, while keeping manifest/runtime behavior local

Shared packages should consume only the contract slices that are relevant to their own domain. They should not become generic pass-through wrappers for the full contract surface.

## Canonical Rules

The current contract defines:

- repo root markers: `.vrooli`, `templates`, `scenarios`, `resources`, `packages`, `cmd`, `internal`, and `go.mod`
- canonical top-level layout under `.vrooli/`, `templates/`, `scenarios/`, `resources/`, `packages/`, `cmd/`, `internal/`, and `docs/`
- canonical scenario manifest path: `scenarios/<name>/.vrooli/service.json`
- optional temporary scenario orientation metadata:
  `scenarios/<name>/.vrooli/orientation.json`
- canonical resource manifest path: `resources/<name>/resource.json`
- repo-aware glob semantics: `doublestar`, root-relative, slash-normalized, case-sensitive, absolute paths rejected
- structural environment variables: `VROOLI_ROOT`, `VROOLI_SOURCE_ROOT`, `VROOLI_SANDBOX_ID`, `VROOLI_SANDBOX_MERGED`, `VROOLI_SANDBOX_SCOPE`
- sandbox full-repo scopes: `""`, `"."`, `"/"`
- named repo-aware profile: `mini_vrooli_bundle`

## Allowed `.vrooli/` Surface

Project-level `.vrooli/` is reserved for repo-owned metadata and one local build output directory.

Allowed tracked entries:

- `.vrooli/repo-contract.json`
- `.vrooli/service.json`
- `.vrooli/schemas/**`
- `.vrooli/resources/**`

Allowed local-only entry:

- `.vrooli/build/**`

Forbidden in project `.vrooli/`:

- secrets and secret files
- runtime state
- logs and telemetry
- deploy targets
- scenario runtime data
- user-specific mutable config
- lockfiles and ad hoc caches

Checked-in scenario behavior or policy files should live in explicit scenario directories such as `config/`, `policy/`, `initiatives/`, `requirements/`, or `docs/`, not in `.vrooli/`, unless they are part of the canonical metadata surface.

Secrets are intentionally outside the repo surface. The canonical shared plaintext store is `~/.vrooli/secrets.json`, scenario-scoped plaintext stores live under `~/.vrooli/scenarios/<scenario>/secrets.json`, and encrypted user secrets live under `~/.vrooli/secrets.enc.json`.

## Compatibility Policy

- The contract describes the future-state Go-native cross-platform structure only.
- Transitional project-level shell layout is explicitly excluded.
- New path migrations that move user or runtime state out of project `.vrooli/` must land greenfield without repo-local compatibility fallbacks.
- Contract changes use semantic versioning:
- `patch`: clarifications and non-breaking metadata
- `minor`: additive fields and additive paths
- `major`: semantic or structural breaks

## Explicit Exclusions

These do not belong in the contract:

- project-level `cli/`
- `cli/commands/`
- `cli/lib/`
- `scripts/lib/`
- `scripts/manage.sh`
- `.git` as a repo-root marker
- `pnpm-workspace.yaml` as a repo-root marker
- `$HOME/Vrooli` fallback semantics
- `APP_ROOT` as a canonical repo-root variable
- scenario-private paths such as `coverage/`, logs, prompts, profiles, queues, or research folders
- project-level secrets paths such as `.vrooli/secrets.json` or `.vrooli/secrets.enc.json`

## Validation

Use the direct CLI for local validation or the Make target for CI/automation:

```bash
vrooli contract validate
make hygiene
```

The `vrooli contract ...` commands are the operator/developer-facing inspection
surface. `make hygiene` remains the CI/automation entrypoint and includes
contract validation plus repository hygiene checks.

Validation currently covers:

- JSON schema compilation
- contract instance validation against the schema
- live repo conformance tests
- explicit checks that excluded legacy paths do not appear in the contract
- semantic drift checks for profile roots, required markers, and canonical path/value invariants

## CLI Tooling

The top-level CLI now exposes the contract directly:

```bash
vrooli contract validate
vrooli contract show
vrooli contract resolve scenario <name> --file service
vrooli contract match-glob <pattern> <path>
```

Use `--json` with each command for machine-readable output.

- `vrooli contract validate` runs the schema validator plus in-process semantic and live drift checks
- `vrooli contract show` prints the loaded contract, root, and the current policy surface
- `vrooli contract resolve scenario <name> --file <key>` resolves canonical scenario paths from the contract
- `vrooli contract match-glob <pattern> <path>` evaluates a path against the contract-defined root-relative glob semantics

Common example: `vrooli contract resolve scenario <name> --file service`

## Adoption Rules

For future repo-aware work:

- do not add new independent repo-root detection logic
- do not add new hard-coded canonical scenario path assembly
- do not introduce new repo-aware glob semantics outside the shared contract path
- do not treat historical fallbacks as future-state architecture
- when changing the contract, update the schema, docs, and `internal/repocontract` coverage in the same change
- add a new structural rule to the contract only if it is intentionally shared, future-state aligned, and stable enough to version

Ordinary scenario runtime logic should usually consume higher-level shared packages. Repo-aware infrastructure code may consume `packages/repo-contract-go` directly when repository/layout semantics are part of the job.

Preferred consumption order:

- ordinary runtime logic should prefer `packages/api-core` or `packages/cli-core`
- repo-aware infrastructure code may consume `packages/repo-contract-go` directly when layout semantics are central to its job
- local fallbacks are acceptable only when they are explicitly documented as non-authoritative compatibility behavior

Phase 6 does not claim that every historical repo-root helper or path join in the monorepo has already been migrated. It closes adoption by making future repo-aware work follow the contract and by treating remaining debt as consumer cleanup rather than as an allowlisted alternative authority.

## Landed Consumer Migrations

These high-risk consumers now have repo-contract-backed primary resolution paths and should stay aligned with the contract:

- `swarm-manager`
- `scenario-to-cloud`
- `tidiness-manager`
- `workspace-sandbox`
- `test-genie`
- `scenario-auditor`
- `git-control-tower`

Residual follow-up work should be treated as consumer cleanup, not as alternative contract authority.

One implementation detail remains intentionally local: `scenario-auditor/api/rules/structure/ui_structure.go` keeps stdlib-only rule-file discovery because the yaegi rule loader used for rule validation cannot resolve transitive third-party imports such as `repo-contract-go`. That rule-local fallback is not contract authority and does not change the shared runtime migration status.
