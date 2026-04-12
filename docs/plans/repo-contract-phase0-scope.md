# Repo Contract Phase 0 Scope

This document records the Phase 0 discovery pass for the repository contract work described in [repo-contract-implementation-plan.md](/home/matthalloran8/Vrooli/docs/plans/repo-contract-implementation-plan.md).

Phase 0 purpose:

- identify which repository/layout assumptions are canonical future-state contract material
- identify which assumptions are transitional legacy behavior
- identify which assumptions are scenario-private and must stay out of the shared contract
- identify the first migration targets whose duplicated layout logic creates the highest risk

This is a boundary-locking artifact. It does not define the final schema or adapter API.

## Classification Rules

- `canonical`
  - intentionally shared across the repo
  - aligned with the future-state Go-native project structure
  - stable enough to appear in `.vrooli/repo-contract.json`
- `legacy`
  - still present in implementation code
  - transitional compatibility behavior
  - must not be normalized into the future-state contract
- `private`
  - real and sometimes important, but owned by one scenario or implementation
  - not a stable repo-wide assumption

## Authoritative Inputs

Higher-authority sources for this classification:

- [project-level-bash-to-go-migration-plan.md](/home/matthalloran8/Vrooli/docs/plans/project-level-bash-to-go-migration-plan.md)
- [resource-cross-platform-migration-plan.md](/home/matthalloran8/Vrooli/docs/plans/resource-cross-platform-migration-plan.md)
- current Go-native project layout at repo root:
  - `go.mod`
  - `cmd/`
  - `internal/`
  - `.vrooli/`
  - `scenarios/`
  - `resources/`
  - `packages/`

## Scope Inventory

### Root Detection

| Assumption | Classification | Why |
|---|---|---|
| Repo root contains `go.mod` | `canonical` | Current Go-native project root marker and used by `internal/buildinfo` source-root discovery. |
| Repo root contains `.vrooli/`, `cmd/`, `internal/`, `packages/`, `scenarios/`, `resources/` | `canonical` | Matches the future-state project layout described in the migration plans. |
| `VROOLI_SOURCE_ROOT` identifies the source checkout root | `canonical` | Already part of project-level root resolution and stale-check behavior. |
| `VROOLI_ROOT` identifies the effective runtime repo root | `canonical` | Already exported by the Go CLI and consumed by shared/runtime code. |
| Root detection by `.git` | `legacy` | Works in some tools today, but it is weaker than the future-state layout contract and can match the wrong thing. |
| Root detection by `pnpm-workspace.yaml` | `legacy` | Transitional heuristic in `test-genie`, not a future-state contract marker. |
| Root detection by `$HOME/Vrooli` fallback | `legacy` | Common current fallback, but tied to historical local installation convention rather than canonical structure. |
| Shallow ancestor walking with arbitrary depth limits | `legacy` | Implementation detail in some consumers, not a contract rule. |

### Top-Level Layout

| Assumption | Classification | Why |
|---|---|---|
| `.vrooli/` is the project config dir | `canonical` | Stable project-level configuration root. |
| `scenarios/` is the canonical scenario root | `canonical` | Shared across project-level and scenario-level tooling. |
| `resources/` is the canonical resource root | `canonical` | Shared across resource control-plane planning and active resource implementations. |
| `packages/` is the canonical shared package root | `canonical` | Stable part of the Go-native repo structure. |
| `cmd/` is the canonical project command root | `canonical` | Stable part of the Go-native repo structure. |
| `internal/` is the canonical project internal-code root | `canonical` | Stable part of the Go-native repo structure. |
| `docs/` is a canonical top-level docs root | `canonical` | Stable and low-risk top-level directory assumption. |
| Top-level `api/`, `src/`, `scripts/`, `platforms/`, `assets/` | `legacy` | Currently used by `scenario-to-cloud` bundle logic, but not part of the future-state root contract. |
| Top-level `cli/` and project shell helper paths | `legacy` | Explicitly excluded by the main implementation plan. |

### Canonical Scenario Layout

| Assumption | Classification | Why |
|---|---|---|
| Scenario root is `scenarios/<name>` | `canonical` | Shared assumption across the repo and already implemented in `internal/scenario`. |
| Scenario existence is determined by `scenarios/<name>/.vrooli/service.json` | `canonical` | This is the current discovery rule in the Go-native `internal/scenario` package. |
| `.vrooli/service.json` is the canonical shared scenario manifest path | `canonical` | Used across the platform and already part of shared logic. |
| `api`, `ui`, `cli`, `docs`, `requirements`, `initialization` are well-known scenario subpaths | `canonical` | These are cross-scenario conventions already referenced by repo-aware tooling. |
| `.vrooli/metadata.json` is a universally shared scenario file | `private` | Some scenarios use it, but it is not yet a stable repo-wide invariant. |
| `coverage/` is a canonical scenario path | `private` | Used by specific tools like `test-genie`, but not a stable shared structural contract. |
| Scenario-private logs, research, queue, profiles, prompts, etc. | `private` | Real folders, but not shared platform assumptions. |

### Canonical Resource Layout

| Assumption | Classification | Why |
|---|---|---|
| Resource root is `resources/<name>` | `canonical` | Stable future-state root for active resources. |
| Resource manifest path is `resources/<name>/resource.json` | `canonical` | This is the current live manifest location for implemented resources. |
| Resource `docs/` and `initialization/` are well-known subpaths where present | `canonical` | Stable enough to carry as low-risk optional well-known paths. |
| `.vrooli/resource.json` is the resource manifest path | `legacy` | Proposed in the draft shape, but not reflected by current live resources. |
| Resource shell entrypoints such as `cli.sh`, `config/defaults.sh`, `lib/*.sh` | `legacy` | Transitional implementation details covered by the resource migration plan, not future-state contract. |

### Sandbox and Environment Variables

| Assumption | Classification | Why |
|---|---|---|
| `VROOLI_SANDBOX_ID` | `canonical` | Shared sandbox identity env var across sandbox-aware tools. |
| `VROOLI_SANDBOX_MERGED` | `canonical` | Shared overlay mount root for sandbox-aware path resolution. |
| `VROOLI_SANDBOX_SCOPE` | `canonical` | Shared sandbox scope contract already duplicated in multiple Go packages. |
| Full-repo sandbox scope is `""`, `"."`, or `"/"` | `canonical` | Shared semantic already encoded in current sandbox logic. |
| Scenario scope semantics rooted under `scenarios/<name>` | `canonical` | Shared sandbox meaning already present in `internal/scenario` and `cli-core`. |
| `APP_ROOT` as a fallback repo-root contract variable | `legacy` | Seen in some scenario code, but not part of the future-state cross-platform platform contract. |

### Glob Semantics

| Assumption | Classification | Why |
|---|---|---|
| Repo-aware globs are root-relative | `canonical` | This is the intended meaning in the implementation plan and the highest-value shared rule. |
| Absolute paths are rejected | `canonical` | Safety rule already implied by existing validation logic. |
| One glob engine and one syntax policy must be used for both validation and execution | `canonical` | Required to eliminate current semantic drift. |
| `doublestar` semantics are the current likely v1 baseline | `canonical` | Existing execution paths already depend on `**` behavior. |
| `filepath.Match` syntax as validation-only behavior | `legacy` | This is an active source of inconsistency in `swarm-manager`. |

### Bundle and Deploy Profiles

| Assumption | Classification | Why |
|---|---|---|
| Named repo-aware profiles belong in the contract | `canonical` | The implementation plan explicitly targets this. |
| Include/exclude lists for `scenario-to-cloud` mini bundles are a first migration target | `canonical` | Repo-aware bundle composition is a valid direct consumer of the contract. |
| `scenario-to-cloud` current hard-coded include roots are themselves canonical | `legacy` | They are evidence for Phase 0, but should not be copied into the contract without refinement. |

## Phase 0 Findings

### Findings That Should Shape Phase 1

- The initial draft shape should be corrected so resource manifests live at `resources/<name>/resource.json`, not `.vrooli/resource.json`.
- The contract should distinguish `VROOLI_SOURCE_ROOT` and `VROOLI_ROOT`; they are both shared env vars, but they serve different roles.
- `.vrooli/metadata.json` should be treated as optional or deferred until there is clearer repo-wide adoption.
- Root detection in the future adapter must be stricter than current ad hoc fallbacks. Phase 1 should prefer future-state structural markers over generic workspace heuristics.

### Findings That Must Stay Out Of Scope

- storage policy
- lifecycle execution behavior
- health-check behavior
- language/toolchain-specific build details
- scenario-private folder structures

## First High-Risk Consumers

The first migration targets identified during Phase 0 are listed in:

- [repo-contract-phase0-migration-targets.md](/home/matthalloran8/Vrooli/docs/plans/repo-contract-phase0-migration-targets.md)

