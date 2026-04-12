# Repo Contract Phase 4 Implementation Checklist

**Status:** In Progress
**Parent plan:** [repo-contract-implementation-plan.md](/home/matthalloran8/Vrooli/docs/plans/repo-contract-implementation-plan.md)
**Goal:** Fully migrate the remaining high-risk repo-aware consumers onto the shared repo contract and remove duplicated future-state layout logic.

## Exit Criteria

- [x] No Phase 4 consumer validates repo globs outside `packages/repo-contract-go`
- [ ] No Phase 4 consumer derives repo root from `$HOME/Vrooli`, `.git`, `pnpm-workspace.yaml`, or handler-relative path climbing
- [ ] No Phase 4 consumer hard-codes canonical `scenarios/<name>` or `.vrooli/service.json` paths where contract helpers already cover them
- [ ] Each Phase 4 consumer has a single authoritative repo/scenario resolution helper path rather than parallel legacy fallbacks
- [ ] Targeted tests assert repo-contract semantics rather than legacy behavior
- [ ] Focused validation commands pass for every migrated consumer

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
- [ ] Checklist-level test coverage still incomplete

Files:
- [scenarios/tidiness-manager/api/services.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/services.go)
- [scenarios/tidiness-manager/api/smart_scanner.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/smart_scanner.go)
- [scenarios/tidiness-manager/api/smart_scanner_test.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/smart_scanner_test.go)

Tasks:
- [x] Replace `$HOME/Vrooli` and raw `VROOLI_ROOT` fallbacks with repo-contract root resolution
- [x] Replace scenario path construction with repo-contract helpers
- [x] Preserve explicit absolute-path overrides for sandbox/agent callers
- [ ] Add explicit tests for repo discovery failure and absolute-path override behavior
- [ ] Confirm broader package test expectations or narrow validation target in the checklist if unrelated failures remain

Validation:
- [ ] `go test ./...`

## 4. `workspace-sandbox`

Status:
- [x] Core runtime migration landed
- [ ] Contract-backed test coverage still thin

Files:
- [scenarios/workspace-sandbox/api/internal/toolexecution/executor.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/toolexecution/executor.go)
- [scenarios/workspace-sandbox/api/main.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/main.go)
- [scenarios/workspace-sandbox/api/internal/config/config.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/config/config.go)
- [scenarios/workspace-sandbox/api/internal/sandbox/service.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/sandbox/service.go)

Tasks:
- [x] Replace `$HOME/Vrooli` project-root fallback with repo-contract-backed resolution
- [x] Resolve the `workspace-sandbox` scenario directory via the contract for profile store initialization
- [x] Consolidate default project-root discovery to one helper
- [ ] Add tests for contract-backed default root resolution and explicit override precedence
- [ ] Update user-facing/toolregistry text that still claims `VROOLI_ROOT` is the default project-root source

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
- [ ] Primary standards/store path migration landed
- [ ] Parallel legacy repo-root helpers still remain

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
- [ ] Remove the parallel legacy `getVrooliRoot` / `resolveScenarioPath` path from Claude, issue-tracker, and agent-manager flows
- [ ] Decide whether `internal/ruleengine/loader.go` should consume the shared helper or remain a justified bootstrap exception
- [ ] Add/update tests for single-scenario and all-scenarios scans plus the remaining Claude/agent-manager resolution paths

Validation:
- [x] `go test ./...`

## 7. `git-control-tower`

Status:
- [x] Main repo/scenario/service resolution migration landed
- [ ] Scope-prefix and handler test cleanup still remain

Files:
- [scenarios/git-control-tower/api/git_runner_core.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/git_runner_core.go)
- [scenarios/git-control-tower/api/scenario_envelope.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/scenario_envelope.go)
- [scenarios/git-control-tower/api/tidiness_manager_handler.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/tidiness_manager_handler.go)
- [scenarios/git-control-tower/api/review_handler.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/review_handler.go)

Tasks:
- [x] Replace repo-root discovery internals with repo-contract-backed resolution
- [x] Replace canonical scenario/service path joins with repo-contract helpers
- [x] Keep Git validation and active-repo selection logic in `RepoService`
- [ ] Replace remaining hard-coded `scenarios/%s/` scope-prefix generation with contract-backed sandbox scope helpers
- [ ] Add tests for contract-backed scenario resolution in tidiness/review paths, not just envelope

Validation:
- [x] `go test ./...`

## Execution Order

- [x] 1. `swarm-manager`
- [x] 2. `scenario-to-cloud`
- [ ] 3. `scenario-auditor`
- [ ] 4. `git-control-tower`
- [ ] 5. `workspace-sandbox`
- [ ] 6. `tidiness-manager`
- [x] 7. `test-genie`
