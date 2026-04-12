# Repo Contract Phase 4 Implementation Checklist

**Status:** In Progress
**Parent plan:** [repo-contract-implementation-plan.md](/home/matthalloran8/Vrooli/docs/plans/repo-contract-implementation-plan.md)
**Goal:** Fully migrate the remaining high-risk repo-aware consumers onto the shared repo contract and remove duplicated future-state layout logic.

## Exit Criteria

- [ ] No Phase 4 consumer validates repo globs outside `packages/repo-contract-go`
- [ ] No Phase 4 consumer derives repo root from `$HOME/Vrooli`, `.git`, `pnpm-workspace.yaml`, or handler-relative path climbing
- [ ] No Phase 4 consumer hard-codes canonical `scenarios/<name>` or `.vrooli/service.json` paths where contract helpers already cover them
- [ ] Targeted tests assert repo-contract semantics rather than legacy behavior
- [ ] Focused validation commands pass for every migrated consumer

## 1. `swarm-manager`

Files:
- [scenarios/swarm-manager/api/internal/backlog/types.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/types.go)
- [scenarios/swarm-manager/api/internal/backlog/validate_globs.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/validate_globs.go)
- [scenarios/swarm-manager/api/internal/backlog/types_test.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/types_test.go)
- [scenarios/swarm-manager/api/internal/backlog/validate_globs_test.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/validate_globs_test.go)

Tasks:
- [ ] Replace `filepath.Match` glob validation with `repocontract.ValidateRepoGlob`
- [ ] Replace handler-side `doublestar.FilepathGlob` counting with `repocontract.FileMatchCount`
- [ ] Replace handler-relative repo-root derivation with repo-contract root resolution
- [ ] Add regression coverage for `**`, `./`, absolute-path rejection, and parent traversal rejection

Validation:
- [ ] `go test ./api/internal/backlog/...`

## 2. `scenario-to-cloud`

Files:
- [scenarios/scenario-to-cloud/api/bundle/builder.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/bundle/builder.go)
- [scenarios/scenario-to-cloud/api/bundling_rules_test.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/bundling_rules_test.go)
- [scenarios/scenario-to-cloud/api/freshness.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/freshness.go)
- [scenarios/scenario-to-cloud/api/scenarios.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/scenarios.go)

Tasks:
- [ ] Replace bespoke include/exclude policy with `mini_vrooli_bundle` contract profile resolution
- [ ] Keep manifest-specific augmentation in code only
- [ ] Replace canonical scenario/service path assembly with repo-contract helpers where appropriate
- [ ] Update bundle tests to assert contract-backed roots and excludes

Validation:
- [ ] `go test ./api/...`

## 3. `tidiness-manager`

Files:
- [scenarios/tidiness-manager/api/services.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/services.go)
- [scenarios/tidiness-manager/api/smart_scanner.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/smart_scanner.go)
- [scenarios/tidiness-manager/api/smart_scanner_test.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/smart_scanner_test.go)

Tasks:
- [ ] Replace `$HOME/Vrooli` and raw `VROOLI_ROOT` fallbacks with repo-contract root resolution
- [ ] Replace scenario path construction with repo-contract helpers
- [ ] Preserve explicit absolute-path overrides for sandbox/agent callers
- [ ] Add tests for repo discovery, contract failure, and override behavior

Validation:
- [ ] `go test ./api/...`

## 4. `workspace-sandbox`

Files:
- [scenarios/workspace-sandbox/api/internal/toolexecution/executor.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/toolexecution/executor.go)
- [scenarios/workspace-sandbox/api/main.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/main.go)
- [scenarios/workspace-sandbox/api/internal/config/config.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/config/config.go)
- [scenarios/workspace-sandbox/api/internal/sandbox/service.go](/home/matthalloran8/Vrooli/scenarios/workspace-sandbox/api/internal/sandbox/service.go)

Tasks:
- [ ] Replace `$HOME/Vrooli` project-root fallback with repo-contract-backed resolution
- [ ] Resolve the `workspace-sandbox` scenario directory via the contract for profile store initialization
- [ ] Consolidate default project-root discovery to one helper
- [ ] Add tests for default root resolution and explicit override precedence

Validation:
- [ ] `go test ./api/...`

## 5. `test-genie`

Files:
- [scenarios/test-genie/cli/internal/repo/detect.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/internal/repo/detect.go)
- [scenarios/test-genie/cli/internal/repo/detect_test.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/internal/repo/detect_test.go)
- [scenarios/test-genie/cli/execute/report/printer.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/execute/report/printer.go)
- [scenarios/test-genie/cli/execute/report/artifacts.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/execute/report/artifacts.go)

Tasks:
- [ ] Replace `.git` / `pnpm-workspace.yaml` root heuristics with repo-contract root resolution
- [ ] Keep `coverage/` discovery scenario-local
- [ ] Preserve sandbox-aware scenario execution behavior already handled by `cli-core`
- [ ] Update tests to use contract markers instead of `.git`

Validation:
- [ ] `go test ./cli/... ./api/...`

## 6. `scenario-auditor`

Files:
- [scenarios/scenario-auditor/api/handlers_standards.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_standards.go)
- [scenarios/scenario-auditor/api/handlers_scanner.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_scanner.go)
- [scenarios/scenario-auditor/api/handlers_rules.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/handlers_rules.go)
- [scenarios/scenario-auditor/api/standards_store.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/standards_store.go)
- [scenarios/scenario-auditor/api/vulnerability_store.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/vulnerability_store.go)
- [scenarios/scenario-auditor/api/protected_scenarios_store.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/protected_scenarios_store.go)

Tasks:
- [ ] Replace `VROOLI_ROOT` / `$HOME/Vrooli` scenario-root helpers with repo-contract-backed resolution
- [ ] Enumerate “all scenarios” from the contract-defined scenario directory
- [ ] Preserve explicit sandbox path override behavior for agent-driven scans
- [ ] Add/update tests for single-scenario and all-scenarios scans

Validation:
- [ ] `go test ./api/...`

## 7. `git-control-tower`

Files:
- [scenarios/git-control-tower/api/git_runner_core.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/git_runner_core.go)
- [scenarios/git-control-tower/api/scenario_envelope.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/scenario_envelope.go)
- [scenarios/git-control-tower/api/tidiness_manager_handler.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/tidiness_manager_handler.go)
- [scenarios/git-control-tower/api/review_handler.go](/home/matthalloran8/Vrooli/scenarios/git-control-tower/api/review_handler.go)

Tasks:
- [ ] Replace repo-root discovery internals with repo-contract-backed resolution
- [ ] Replace canonical scenario/service path joins with repo-contract helpers
- [ ] Keep Git validation and active-repo selection logic in `RepoService`
- [ ] Add tests for contract-backed scenario resolution in envelope/tidiness/review paths

Validation:
- [ ] `go test ./api/...`

## Execution Order

- [ ] 1. `swarm-manager`
- [ ] 2. `scenario-to-cloud`
- [ ] 3. `tidiness-manager`
- [ ] 4. `workspace-sandbox`
- [ ] 5. `test-genie`
- [ ] 6. `scenario-auditor`
- [ ] 7. `git-control-tower`
