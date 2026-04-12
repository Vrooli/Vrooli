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

- [.vrooli/repo-contract.json](/home/matthalloran8/Vrooli/.vrooli/repo-contract.json)
- [.vrooli/schemas/repo-contract.schema.json](/home/matthalloran8/Vrooli/.vrooli/schemas/repo-contract.schema.json)

## Phase 1 Scope

Phase 1 defines and validates the contract. It does not yet require every consumer to use a language adapter.

Phase 1 implementation status:

- the versioned contract is landed
- the schema is landed
- repo conformance and drift tests are landed
- validation entrypoints are landed
- consumer migration remains deferred to later phases

Phase 1 includes:

- a versioned JSON contract
- a JSON schema for that contract
- versioning and compatibility rules
- normalization rules for slash-style repo-relative paths
- conformance tests against the live repo
- validation entrypoints suitable for CI

Phase 1 excludes:

- `packages/repo-contract-go`
- broad consumer migration
- new runtime behavior beyond validation and documentation

## Phase 1 Completion Rules

Phase 1 should be considered complete only when all of the following remain true:

- `.vrooli/repo-contract.json` stays aligned with the future-state repo shape
- `.vrooli/schemas/repo-contract.schema.json` enforces the current contract shape
- `make validate-repo-contract` remains the single documented validation entrypoint
- `internal/repocontract` catches schema drift, semantic drift, and legacy-path regressions
- deferred consumers are clearly documented as migration targets rather than contract authority

## Canonical Rules

The current contract defines:

- repo root markers: `.vrooli`, `scenarios`, `resources`, `packages`, `cmd`, `internal`, and `go.mod`
- canonical top-level layout under `.vrooli/`, `scenarios/`, `resources/`, `packages/`, `cmd/`, `internal/`, and `docs/`
- canonical scenario manifest path: `scenarios/<name>/.vrooli/service.json`
- canonical resource manifest path: `resources/<name>/resource.json`
- repo-aware glob semantics: `doublestar`, root-relative, slash-normalized, case-sensitive, absolute paths rejected
- structural environment variables: `VROOLI_ROOT`, `VROOLI_SOURCE_ROOT`, `VROOLI_SANDBOX_ID`, `VROOLI_SANDBOX_MERGED`, `VROOLI_SANDBOX_SCOPE`
- sandbox full-repo scopes: `""`, `"."`, `"/"`
- named repo-aware profile: `mini_vrooli_bundle`

## Compatibility Policy

- The contract describes the future-state Go-native cross-platform structure only.
- Transitional project-level shell layout is explicitly excluded.
- Consumers may keep temporary fallback behavior while migrating, but those fallbacks do not change contract semantics.
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

## Validation

Use either of these:

```bash
python3 .vrooli/schemas/validate-repo-contract.py
make validate-repo-contract
```

Validation currently covers:

- JSON schema compilation
- contract instance validation against the schema
- live repo conformance tests
- explicit checks that excluded legacy paths do not appear in the contract
- semantic drift checks for profile roots, required markers, and canonical path/value invariants

## Adoption Rules

For covered repo-aware work:

- do not add new independent repo-root detection logic
- do not add new hard-coded canonical scenario path assembly
- do not introduce new repo-aware glob semantics outside the shared contract path
- do not treat historical fallbacks as future-state architecture
- when changing the contract, update the schema, docs, and `internal/repocontract` coverage in the same change
- add a new structural rule to the contract only if it is intentionally shared, future-state aligned, and stable enough to version

Ordinary scenario runtime logic should usually consume higher-level shared packages. Repo-aware infrastructure code may consume a future adapter directly once Phase 2 lands.

## Known Deferred Consumers

These are still migration targets and should not be treated as Phase 1 precedent:

- `swarm-manager` backlog glob validation and counting
- `scenario-to-cloud` bundle root/include policy
- `tidiness-manager` scenario location fallback logic
- `test-genie` CLI repo-root detection
- `packages/cli-core` repo-root fallback behavior

Their current behavior is legacy compatibility, not contract authority.
