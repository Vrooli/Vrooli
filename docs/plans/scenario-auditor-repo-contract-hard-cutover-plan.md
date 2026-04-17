# Scenario-Auditor Repo Contract Hard Cutover Plan

## Status

Proposed on 2026-04-16.

This plan is a greenfield hard cutover for `scenario-auditor` repo-aware behavior. It replaces the current mixed migration state with a single repo-contract authority path for runtime code and tests, then removes the old helpers and stale test scaffolding entirely.

## Goal

Make `scenario-auditor` fully and cleanly repo-contract-native:

- one authoritative repo/scenario resolution model
- no duplicate root/scenario helper paths
- no ambient test dependence on ad hoc cwd/env state except where explicitly under test
- no hand-authored stale repo-contract fixture JSON scattered across tests
- no retained fallback/legacy/compatibility paths after cutover

This is not a fixture patch. It is a structural cleanup and hard migration.

## Problem Statement

`scenario-auditor` predates the repo-contract system. Its current code reflects incremental migration rather than a clean redesign around the contract.

Today the scenario has:

- app-level repo-contract wrappers in [scenarios/scenario-auditor/api/repo_contract.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/repo_contract.go)
- cached global root helpers in [scenarios/scenario-auditor/api/agent_manager.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/agent_manager.go)
- separate rule-engine repo discovery in [scenarios/scenario-auditor/api/internal/ruleengine/loader.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/internal/ruleengine/loader.go)
- stale hand-authored repo-contract fixtures in [scenarios/scenario-auditor/api/summary_test.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/summary_test.go)

This creates multiple failure modes:

- schema drift in test fixtures, such as missing `layout.template_dir`
- global cached path state that must be manually reset in tests
- duplicate resolution logic with different behavior across runtime surfaces
- unclear separation between “resolve repo layout” and “load rules / persist artifacts / build prompts”

## Root Cause

The root cause is architectural drift:

1. `scenario-auditor` originally owned its own repo/path assumptions.
2. repo-contract became the platform authority later.
3. `scenario-auditor` adopted repo-contract incrementally instead of cutting over to one explicit contract-backed context model.

The immediate failing tests are symptoms of that drift, not the core problem.

## Architectural Decision

Treat repo-contract as the sole authority for repo-aware behavior inside `scenario-auditor`, and route all repo/scenario path resolution through one explicit internal context object.

Do not preserve dual behavior where:

- some code calls repo-contract directly
- some code uses cached wrapper globals
- some code rediscovers from cwd/env on demand
- some tests handcraft partial contracts and hope they still validate

After this migration, there should be exactly one sanctioned pattern:

- build a `RepoContext`
- pass or access that context deliberately
- resolve repo/scenario paths through it

## Target End State

### Runtime

- `scenario-auditor` has one internal repo-resolution package, tentatively:
  - `scenarios/scenario-auditor/api/internal/repocontext`
- that package owns:
  - repo root
  - loaded contract
  - scenarios root
  - scenario-auditor root
  - `ResolveScenarioPath(name)`
  - `RelativeToRepoRoot(path)`
- app/runtime code no longer calls `repocontract.FindRepoRootFromEnvOrCWD`, `repocontract.LoadDefault`, or `repocontract.ResolveScenarioPath` directly outside that package

### Tests

- all repo-aware tests use one canonical fixture/harness builder
- the harness always emits a fully valid repo-contract file
- fixture construction is centralized and reusable
- only a narrow dedicated test set uses env/cwd fallback semantics

### Cleanup

- old wrappers are removed
- cached path globals are removed
- test-only reset helpers for those globals are removed
- duplicated inline fixture contract JSON is removed

## Scope

### In scope

- `scenario-auditor/api` runtime path resolution
- `scenario-auditor/api/internal/ruleengine` repo/rule directory resolution
- repo-aware tests in `scenario-auditor/api`
- associated docs/comments/tests that still describe the old resolution model

### Out of scope

- the separate `ui_lifecycle_launch` rule-loader compatibility bug
- external provider noise during broader `scenario-auditor` tests
- other scenarios that consume repo-contract unless they must adapt to a changed `scenario-auditor` public contract

## Files Likely Affected

Primary runtime files:

- [scenarios/scenario-auditor/api/repo_contract.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/repo_contract.go)
- [scenarios/scenario-auditor/api/agent_manager.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/agent_manager.go)
- [scenarios/scenario-auditor/api/handlers_claude.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_claude.go)
- [scenarios/scenario-auditor/api/handlers_issue_tracker.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_issue_tracker.go)
- [scenarios/scenario-auditor/api/handlers_standards.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_standards.go)
- [scenarios/scenario-auditor/api/main.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/main.go)
- [scenarios/scenario-auditor/api/rule_service.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/rule_service.go)
- [scenarios/scenario-auditor/api/summary.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/summary.go)
- [scenarios/scenario-auditor/api/internal/ruleengine/loader.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/internal/ruleengine/loader.go)

Primary test files:

- [scenarios/scenario-auditor/api/summary_test.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/summary_test.go)
- [scenarios/scenario-auditor/api/handlers_standards_test.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_standards_test.go)
- [scenarios/scenario-auditor/api/handlers_claude_split_test.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_claude_split_test.go)
- [scenarios/scenario-auditor/api/handlers_rules_coverage_test.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_rules_coverage_test.go)
- [scenarios/scenario-auditor/api/rule_loader_all_rules_test.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/rule_loader_all_rules_test.go)

New files expected:

- `scenarios/scenario-auditor/api/internal/repocontext/...`
- `scenarios/scenario-auditor/api/internal/testrepo/...` or equivalent shared repo-fixture harness

## Phases

## Phase 0: Baseline And Lock The Contract

Purpose:
- establish the exact target behavior before implementation

Tasks:
- [ ] Enumerate every repo-aware entrypoint in `scenario-auditor/api`
- [ ] Group each call site into one of:
  - repo root resolution
  - scenario path resolution
  - scenario-auditor root resolution
  - repo-relative path formatting
  - rule directory discovery
- [ ] Confirm the canonical repo-contract API surface to use from `packages/repo-contract-go`
- [ ] Record any remaining runtime surfaces that truly require env/cwd fallback rather than explicit context

Exit criteria:
- [ ] Every repo-aware call site is mapped
- [ ] One authoritative contract-backed resolution shape is chosen
- [ ] No unresolved debate remains about whether caches or parallel wrappers survive

## Phase 1: Introduce `RepoContext`

Purpose:
- create the single authority for repo-aware behavior

Tasks:
- [ ] Add `api/internal/repocontext`
- [ ] Define a `Context` or similarly explicit type that owns:
  - resolved repo root
  - loaded contract
  - scenarios root
  - scenario-auditor root
- [ ] Provide constructors:
  - [ ] `FromEnvOrCWD()`
  - [ ] `FromRepoRoot(root string)`
- [ ] Provide methods:
  - [ ] `RepoRoot()`
  - [ ] `ScenarioAuditorRoot()`
  - [ ] `ScenariosRoot()`
  - [ ] `ResolveScenarioPath(name string)`
  - [ ] `RelativeToRepoRoot(path string)`
- [ ] Keep this package thin and deterministic; no hidden global state

Exit criteria:
- [ ] There is one explicit repo-resolution abstraction
- [ ] The abstraction uses repo-contract directly
- [ ] It does not depend on package-global caches

## Phase 2: Rewire Runtime Code

Purpose:
- migrate runtime consumers onto `RepoContext`

Tasks:
- [ ] Replace usage of `resolveRepoRoot`
- [ ] Replace usage of `resolveScenariosRoot`
- [ ] Replace usage of `resolveContractScenarioPath`
- [ ] Replace usage of `resolveScenarioAuditorRoot`
- [ ] Replace usage of `currentVrooliRoot`
- [ ] Replace usage of `getScenarioRoot`
- [ ] Update prompt/template/artifact/store flows to read from `RepoContext`
- [ ] Ensure the scenario main/runtime wiring initializes and reuses one context instance where appropriate

Exit criteria:
- [ ] Runtime consumers no longer bypass `RepoContext`
- [ ] Repo/scenario path semantics are uniform across the app

## Phase 3: Rewire Rule Engine Integration

Purpose:
- remove parallel repo discovery from the rule-engine path

Tasks:
- [ ] Decide the correct boundary:
  - preferred: app code resolves repo root / rule directories, rule-engine consumes explicit inputs
- [ ] Update `internal/ruleengine/loader.go` to stop owning policy-level repo discovery where not needed
- [ ] Keep rule-engine focused on rule loading/execution, not application layout policy
- [ ] Ensure app and rule-engine use the same resolved repo root semantics

Exit criteria:
- [ ] There is no second authoritative repo-resolution path in rule-engine
- [ ] Rule discovery semantics are driven by the same contract-backed context model

## Phase 4: Replace Test Infrastructure With A Canonical Repo Harness

Purpose:
- eliminate stale ad hoc contract fixtures

Tasks:
- [ ] Introduce one shared repo-fixture builder for `scenario-auditor/api` tests
- [ ] Ensure the harness writes a fully valid repo-contract file
- [ ] Include required layout fields such as `layout.template_dir`
- [ ] Include helper methods for:
  - [ ] creating scenarios
  - [ ] creating service manifests
  - [ ] changing cwd safely
  - [ ] building rule directory structures
- [ ] Migrate existing repo-aware tests onto the harness
- [ ] Remove inline contract JSON builders once migrated

Exit criteria:
- [ ] Repo-aware tests no longer handcraft partial contracts
- [ ] Contract schema changes require updating one fixture harness, not many tests

## Phase 5: Narrow And Clarify Discovery Tests

Purpose:
- make tests deterministic and explicit

Tasks:
- [ ] Rewrite most repo-aware tests to use `FromRepoRoot(...)` or explicit fixture roots
- [ ] Keep env/cwd fallback testing only in a dedicated focused test file
- [ ] Remove incidental dependence on process cwd in tests that are not about cwd behavior
- [ ] Remove manual reset helpers that only exist because of old global caches

Exit criteria:
- [ ] Most tests use explicit roots
- [ ] Env/cwd fallback behavior is covered, but isolated
- [ ] No test relies on ambient process state accidentally

## Phase 6: Cleanup Hard Cutover

Purpose:
- remove all old code and migration residue

Tasks:
- [ ] Delete obsolete helpers in `repo_contract.go` if fully replaced
- [ ] Delete `scenarioRootOnce`, `scenarioRootPath`, `vrooliRootOnce`, `vrooliRootPath`
- [ ] Delete any reset helpers that existed only for those globals
- [ ] Delete inline repo-contract fixture JSON builders that are no longer used
- [ ] Remove comments/docs/tests that still describe the old mixed-resolution model
- [ ] Confirm no runtime code still calls repo-contract directly outside the new `repocontext` authority except in the authority package itself

Exit criteria:
- [ ] Old helper path is gone
- [ ] Old cached root path is gone
- [ ] Old fixture path is gone
- [ ] No dead migration code remains

## Phase 7: Validation

Purpose:
- prove the cutover is correct and complete

### Focused tests

- [ ] `go test ./...` in `scenarios/scenario-auditor/api/internal/repocontext/...`
- [ ] focused repo-aware tests in `scenarios/scenario-auditor/api`
- [ ] focused rule-engine tests that cover rule directory resolution after the cutover

### Full scenario-auditor validation

- [ ] `go test ./...` in `scenarios/scenario-auditor/api`
- [ ] investigate and separately classify any remaining failures as:
  - [ ] genuinely fixed by this plan
  - [ ] unrelated pre-existing failures

### Static cleanup validation

- [ ] Search for removed legacy helpers and confirm zero runtime references remain
- [ ] Search for inline `repo-contract.json` fixture strings in `scenario-auditor/api` tests and confirm only the canonical harness remains where appropriate
- [ ] Search for direct `repocontract.*` runtime calls outside the designated authority package and confirm none remain unless explicitly justified

Suggested verification commands:

```bash
cd scenarios/scenario-auditor/api
go test ./...

rg "scenarioRootOnce|vrooliRootOnce|resolveRepoRoot\\(|resolveScenarioAuditorRoot\\(|resolveScenariosRoot\\(" scenarios/scenario-auditor/api -S
rg "repo-contract.json" scenarios/scenario-auditor/api -S
rg "repocontract\\." scenarios/scenario-auditor/api -S
```

## Acceptance Criteria

- [ ] `scenario-auditor` has one authoritative repo-contract-backed resolution path
- [ ] No global cached root/scenario helper state remains
- [ ] Rule-engine no longer carries a parallel layout-policy authority
- [ ] Repo-aware tests use one canonical valid fixture harness
- [ ] Most tests use explicit repo roots rather than ambient cwd/env
- [ ] Cleanup phase removes the old code completely
- [ ] Full `go test ./...` for `scenarios/scenario-auditor/api` is green, or any remaining failures are explicitly proven unrelated and tracked separately

## Risks

1. Over-coupling runtime code to a global singleton context.
Mitigation:
- use explicit construction and injection where possible

2. Rule-engine loader may require some local path knowledge.
Mitigation:
- keep path resolution policy outside rule-engine and pass concrete values in

3. Test harness could become another hidden source of contract drift.
Mitigation:
- keep one canonical harness only
- ensure it reflects the current schema completely

## Non-Goals

- Do not preserve old wrapper APIs “just in case”
- Do not keep compatibility shims for the removed root caches
- Do not keep duplicate fixture builders for convenience
- Do not treat partial contract fixtures as acceptable test shortcuts

## Cleanup Checklist

- [ ] old repo helper functions deleted
- [ ] old cached root globals deleted
- [ ] old reset helpers deleted
- [ ] old inline contract fixtures deleted
- [ ] old docs/comments describing mixed behavior deleted or updated
- [ ] search confirms no dead references remain

## Recommended Execution Order

1. Phase 0
2. Phase 1
3. Phase 2
4. Phase 3
5. Phase 4
6. Phase 5
7. Phase 6
8. Phase 7

## Closeout Requirement

Do not mark this plan complete until:

- the cleanup checklist is fully checked off
- validation is complete
- a final diff review confirms the cutover removed the old path instead of layering new code on top of it
