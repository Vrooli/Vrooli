# Repo Contract Phase 0 Migration Targets

This document records the first migration targets identified during Phase 0 discovery.

Priority is based on:

- likelihood of behavioral drift
- number of duplicated structural assumptions
- blast radius if semantics differ between tools
- closeness to the future contract surface

## Priority 1

### 1. `swarm-manager`

Files:

- [types.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/types.go)
- [validate_globs.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/validate_globs.go)

Why first:

- validation and execution already use different glob semantics
- root resolution is inferred from handler-relative filesystem layout instead of a canonical repo contract
- this is the clearest correctness risk in the current codebase

Current duplicated assumptions:

- globs are relative and not absolute
- `filepath.Match` is acceptable validation syntax
- repo root is `filepath.Dir(filepath.Dir(h.rootDir))`
- match counting should use `doublestar.FilepathGlob`

Migration goal:

- move validation and matching onto one contract-backed glob policy
- resolve repo root from the shared contract instead of handler-relative traversal

### 2. `scenario-to-cloud`

Files:

- [builder.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/bundle/builder.go)

Why next:

- bundle composition is explicitly named as a valid direct contract consumer
- current logic hard-codes include roots, exclude globs, and root-level file assumptions
- current logic partially reflects transitional root structure and could drift from future-state decisions

Current duplicated assumptions:

- which top-level roots should be included in a mini bundle
- which top-level files are well-known bundle inputs
- which exclusion patterns are canonical
- which repo roots should be treated as structural vs incidental

Migration goal:

- replace bespoke include/exclude policy with named contract-backed profiles
- keep scenario-specific manifest augmentation in code, but move shared repo layout policy into the contract layer

### 3. `tidiness-manager`

Files:

- [services.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/services.go)

Why next:

- it bakes in `$HOME/Vrooli` fallback behavior and direct scenario-root construction
- it turns weak repo assumptions into user-facing scan behavior
- it is exactly the kind of repo-aware tool the implementation plan lists as a consumer

Current duplicated assumptions:

- `VROOLI_ROOT` or `$HOME/Vrooli` is enough to locate the repo
- scenarios always live under `<root>/scenarios/<name>`
- direct absolute-path normalization is acceptable for request handling

Migration goal:

- resolve scenario roots via shared contract-backed helpers
- remove fallback behavior that should not survive into future-state semantics

### 4. `test-genie`

Files:

- [detect.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/internal/repo/detect.go)

Why next:

- it implements an independent repo-root detector with weaker markers than the future-state contract
- it caches those results and feeds them into scenario/test path discovery
- it is already called out in the implementation plan as a likely repo-aware consumer

Current duplicated assumptions:

- `.git` and `pnpm-workspace.yaml` are sufficient root markers
- bounded ancestor walking is acceptable root discovery
- `coverage/` is a discoverable scenario test directory

Migration goal:

- replace local root detection with contract-backed root resolution
- leave `coverage/` behavior in tool-specific code unless it becomes a wider shared invariant

## Priority 2

### 5. `packages/cli-core`

Files:

- [sandbox.go](/home/matthalloran8/Vrooli/packages/cli-core/cliutil/sandbox.go)

Why:

- it duplicates sandbox scope semantics already present in `internal/scenario`
- it still includes `$HOME/Vrooli` repo fallback logic
- it is a shared package, so contract-backed cleanup here pays down duplication broadly

Migration goal:

- migrate shared sandbox/repo-root rules onto contract-backed helpers while keeping CLI ergonomics in `cli-core`

### 6. `internal/scenario`

Files:

- [scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go)

Why:

- it is already close to the intended future contract and should become a reference consumer
- it currently contains canonical logic plus some migration-era assumptions in one package

Migration goal:

- separate “contract-backed repo/scenario layout rules” from “scenario manifest and runtime behavior”

## Priority 3

### 7. `app-monitor`, `prd-control-tower`, and other ad hoc repo-root helpers

Examples:

- [app_utils.go](/home/matthalloran8/Vrooli/scenarios/app-monitor/api/services/app_utils.go)
- [main.go](/home/matthalloran8/Vrooli/scenarios/prd-control-tower/api/main.go)

Why:

- these are clear duplication points, but they are less central to the initial contract rollout than the Priority 1 consumers

Migration goal:

- remove local root heuristics after the contract adapter exists and the highest-risk consumers are migrated

## Migration Order Summary

1. `swarm-manager`
2. `scenario-to-cloud`
3. `tidiness-manager`
4. `test-genie`
5. `packages/cli-core`
6. `internal/scenario`
7. remaining ad hoc repo-root helpers

## Phase 0 Exit Criteria For This List

Phase 0 is complete when:

- every target above has an identified duplicated assumption set
- every target above has a clear migration goal
- no additional higher-risk consumer remains undocumented

