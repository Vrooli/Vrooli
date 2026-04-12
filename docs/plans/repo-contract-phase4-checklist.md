# Repo Contract Phase 4 Implementation Checklist

**Status:** Complete
**Parent plan:** [repo-contract-implementation-plan.md](/home/matthalloran8/Vrooli/docs/plans/repo-contract-implementation-plan.md)
**Goal:** Fully migrate the remaining high-risk repo-aware consumers onto the shared repo contract and remove duplicated future-state layout logic.

Implementation note:
- Runtime migration is complete for the Phase 4 consumers.
- One `scenario-auditor` rule implementation (`api/rules/structure/ui_structure.go`) intentionally remains stdlib-only for its rule-file discovery path because the yaegi rule loader cannot resolve transitive third-party imports such as `repo-contract-go`. That rule-local discovery path is not treated as contract authority.

## Exit Criteria

- [x] No Phase 4 consumer validates repo globs outside `packages/repo-contract-go`
- [x] No Phase 4 consumer derives repo root from `$HOME/Vrooli`, `.git`, `pnpm-workspace.yaml`, or handler-relative path climbing
- [x] No Phase 4 consumer hard-codes canonical `scenarios/<name>` or `.vrooli/service.json` paths where contract helpers already cover them
- [x] Each Phase 4 consumer has a single authoritative repo/scenario resolution helper path rather than parallel legacy fallbacks
- [x] Targeted tests assert repo-contract semantics rather than legacy behavior
- [x] Focused validation commands pass for every migrated consumer
- [x] Post-migration residual runtime/help-text heuristics in the Phase 4 consumers have been removed or redirected to contract-backed helpers

## 1. `swarm-manager`

Status:
- [x] Runtime migration complete for Phase 4 scope
- [x] Contract-backed regression coverage added

Files:
- [scenarios/swarm-manager/api/internal/backlog/types.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/types.go)
- [scenarios/swarm-manager/api/internal/backlog/validate_globs.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/validate_globs.go)
- [scenarios/swarm-manager/api/internal/backlog/types_test.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/types_test.go)
- [scenarios/swarm-manager/api/internal/backlog/validate_globs_test.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/validate_globs_test.go)

Tasks:
- [x] Replace `filepath.Match` glob validation with `repocontract.ValidateRepoGlob`
- [x] Replace handler-side `doublestar.FilepathGlob` counting with `repocontract.FileMatchCount`
- [x] Replace handler-relative repo-root derivation with repo-contract root resolution
- [x] Add regression coverage for `**`, `./`, absolute-path rejection, and parent traversal rejection

Validation:
- [x] `go test ./internal/backlog/...`

## 2. `scenario-to-cloud`

Status:
- [x] Bundle/profile migration landed
- [x] Runtime repo/scenario/service helpers consolidated
- [x] Legacy repo-root fallbacks removed from investigation/task/VPS paths
- [x] Contract-backed validation completed

Files:
- [scenarios/scenario-to-cloud/api/bundle/builder.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/bundle/builder.go)
- [scenarios/scenario-to-cloud/api/bundle/storage.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/bundle/storage.go)
- [scenarios/scenario-to-cloud/api/bundling_rules_test.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/bundling_rules_test.go)
- [scenarios/scenario-to-cloud/api/freshness.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/freshness.go)
- [scenarios/scenario-to-cloud/api/scenarios.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/scenarios.go)
- [scenarios/scenario-to-cloud/api/handlers_manifest.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/handlers_manifest.go)
- [scenarios/scenario-to-cloud/api/deployment/manifest_refresh.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/deployment/manifest_refresh.go)
- [scenarios/scenario-to-cloud/api/vps/deploy.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/vps/deploy.go)
- [scenarios/scenario-to-cloud/api/secrets/handlers_local.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/secrets/handlers_local.go)
- [scenarios/scenario-to-cloud/api/tasks/service.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/tasks/service.go)
- [scenarios/scenario-to-cloud/api/investigation/service.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/investigation/service.go)
- [scenarios/scenario-to-cloud/api/handlers_docs.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/handlers_docs.go)

Tasks:
- [x] Replace bespoke include/exclude policy with `mini_vrooli_bundle` contract profile resolution
- [x] Keep manifest-specific augmentation in code only
- [x] Add one shared contract-backed helper surface for repo root, scenario root, scenario service path, and the `scenario-to-cloud` scenario root
- [x] Replace remaining `filepath.Join(repoRoot, "scenarios", scenarioID, ".vrooli", "service.json")` joins with repo-contract helpers
- [x] Replace legacy `.git` / raw `VROOLI_ROOT` repo-root discovery in `vps/deploy.go`
- [x] Replace `$HOME/Vrooli` working-dir fallbacks in investigation/task execution paths
- [x] Replace scenario-to-cloud docs discovery fallback with contract-backed scenario docs resolution
- [x] Update/add tests for scenario/service path helpers, manifest refresh, VPS dependency validation, local secrets resolution, and docs discovery

Validation:
- [x] `go test ./...`

## 3. `tidiness-manager`

Status:
- [x] Core runtime migration landed
- [x] Checklist-level test coverage completed

Files:
- [scenarios/tidiness-manager/api/services.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/services.go)
- [scenarios/tidiness-manager/api/smart_scanner.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/smart_scanner.go)
- [scenarios/tidiness-manager/api/smart_scanner_test.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/smart_scanner_test.go)

Tasks:
- [x] Replace `$HOME/Vrooli` and raw `VROOLI_ROOT` fallbacks with repo-contract root resolution
- [x] Replace scenario path construction with repo-contract helpers
- [x] Preserve explicit absolute-path overrides for sandbox/agent callers
- [x] Add explicit tests for repo discovery failure and absolute-path override behavior
- [x] Confirm broader package test expectations with full package validation

Validation:
- [x] `go test ./...`

## 4. `workspace-sandbox`

Status:
- [x] Core runtime migration landed
- [x] Contract-backed test coverage completed

Files:
- [scenarios/workspace-sandbox/api/internal/toolexecution/executor.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/toolexecution/executor.go)
- [scenarios/workspace-sandbox/api/main.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/main.go)
- [scenarios/workspace-sandbox/api/internal/config/config.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/config/config.go)
- [scenarios/workspace-sandbox/api/internal/sandbox/service.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/sandbox/service.go)

Tasks:
- [x] Replace `$HOME/Vrooli` project-root fallback with repo-contract-backed resolution
- [x] Resolve the `workspace-sandbox` scenario directory via the contract for profile store initialization
- [x] Consolidate default project-root discovery to one helper
- [x] Add tests for contract-backed default root resolution and explicit override precedence
- [x] Update user-facing/toolregistry text that still claims `VROOLI_ROOT` is the default project-root source

Validation:
- [x] `go test ./...`

## 5. `test-genie`

Status:
- [x] Runtime migration complete for Phase 4 scope
- [x] Contract-backed tests added

Files:
- [scenarios/test-genie/cli/internal/repo/detect.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/internal/repo/detect.go)
- [scenarios/test-genie/cli/internal/repo/detect_test.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/internal/repo/detect_test.go)
- [scenarios/test-genie/cli/execute/report/printer.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/execute/report/printer.go)
- [scenarios/test-genie/cli/execute/report/artifacts.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/execute/report/artifacts.go)

Tasks:
- [x] Replace `.git` / `pnpm-workspace.yaml` root heuristics with repo-contract root resolution
- [x] Keep `coverage/` discovery scenario-local
- [x] Preserve sandbox-aware scenario execution behavior already handled by `cli-core`
- [x] Update tests to use contract markers instead of `.git`

Validation:
- [x] `go test ./internal/repo/... ./execute/report/...`

## 6. `scenario-auditor`

Status:
- [x] Primary standards/store path migration landed
- [x] Parallel legacy repo-root helpers removed

Files:
- [scenarios/scenario-auditor/api/handlers_standards.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_standards.go)
- [scenarios/scenario-auditor/api/handlers_scanner.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_scanner.go)
- [scenarios/scenario-auditor/api/handlers_rules.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_rules.go)
- [scenarios/scenario-auditor/api/standards_store.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/standards_store.go)
- [scenarios/scenario-auditor/api/vulnerability_store.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/vulnerability_store.go)
- [scenarios/scenario-auditor/api/protected_scenarios_store.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/protected_scenarios_store.go)
- [scenarios/scenario-auditor/api/handlers_claude.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_claude.go)
- [scenarios/scenario-auditor/api/agent_manager.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/agent_manager.go)
- [scenarios/scenario-auditor/api/handlers_issue_tracker.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_issue_tracker.go)
- [scenarios/scenario-auditor/api/internal/ruleengine/loader.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/internal/ruleengine/loader.go)

Tasks:
- [x] Replace `VROOLI_ROOT` / `$HOME/Vrooli` scenario-root helpers in standards/store flows with repo-contract-backed resolution
- [x] Enumerate “all scenarios” from the contract-defined scenario directory
- [x] Preserve explicit sandbox path override behavior for agent-driven scans
- [x] Remove the parallel legacy `getVrooliRoot` / `resolveScenarioPath` path from Claude, issue-tracker, and agent-manager flows
- [x] Move `internal/ruleengine/loader.go` onto repo-contract-backed repo/scenario discovery
- [x] Add/update tests for single-scenario and all-scenarios scans plus the remaining Claude/agent-manager resolution paths

Validation:
- [x] `go test ./...`

## 7. `git-control-tower`

Status:
- [x] Main repo/scenario/service resolution migration landed
- [x] Scope-prefix and handler test cleanup completed

Files:
- [scenarios/git-control-tower/api/git_runner_core.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/git_runner_core.go)
- [scenarios/git-control-tower/api/scenario_envelope.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/scenario_envelope.go)
- [scenarios/git-control-tower/api/tidiness_manager_handler.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/tidiness_manager_handler.go)
- [scenarios/git-control-tower/api/review_handler.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/review_handler.go)

Tasks:
- [x] Replace repo-root discovery internals with repo-contract-backed resolution
- [x] Replace canonical scenario/service path joins with repo-contract helpers
- [x] Keep Git validation and active-repo selection logic in `RepoService`
- [x] Replace remaining hard-coded `scenarios/%s/` scope-prefix generation with contract-backed sandbox scope helpers
- [x] Add tests for contract-backed scenario resolution in tidiness/review paths, not just envelope

Validation:
- [x] `go test ./...`

## Execution Order

- [x] 1. `swarm-manager`
- [x] 2. `scenario-to-cloud`
- [x] 3. `scenario-auditor`
- [x] 4. `git-control-tower`
- [x] 5. `workspace-sandbox`
- [x] 6. `tidiness-manager`
- [x] 7. `test-genie`

## Closeout Notes

- `docs/repo-contract.md` now reflects the landed Phase 4 migrations instead of listing these consumers as deferred.
- `tidiness-manager/api` now passes full `go test ./...`; the earlier targeted-test caveat no longer applies.
- Consumer closeout included residual surfaces outside the original file lists where they still taught or implemented legacy repo-root/path behavior.
- `workspace-sandbox` no longer falls back to `cwd/scenarios/workspace-sandbox/...` for schema discovery.
- `scenario-to-cloud` and `git-control-tower` residual operator/help-text path guidance now avoids teaching `~/Vrooli` or raw `scenarios/<name>` as the canonical model.
- `scenario-auditor` runtime handlers/stores/providers now use the shared repo-contract helpers; only the yaegi-constrained `ui_structure` rule keeps a local discovery fallback.
