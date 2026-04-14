# Repo Contract Phase 6 Checklist

**Status:** In progress
**Parent plan:** [repo-contract-implementation-plan.md](/home/matthalloran8/Vrooli/docs/plans/repo-contract-implementation-plan.md)
**Goal:** Make the repo contract govern future repo-aware work through docs, contributor rules, agent guidance, validation, and enforcement of adoption rules without exception-based escape hatches.

## Exit Criteria

- [x] `docs/repo-contract.md` explains what Phase 6 governs and what it does not
- [x] contributor guidance points repo-aware work at the contract instead of ad hoc root/path heuristics
- [x] agent guidance blocks new independent repo-root detection and canonical path assembly
- [x] agent-facing prompt-manager guidance stops teaching direct `VROOLI_ROOT` + `scenarios/...` joins as the model
- [x] `vrooli contract validate` enforces adoption-rule alignment
- [x] adoption-rule validation runs without an exception allowlist
- [x] remaining migration debt is tracked as follow-up work, not as validator-backed exceptions
- [x] at least one post-Phase-6 migration slice reduces remaining debt

## Implemented

- [x] Added adoption rules to [AGENTS.md](/home/matthalloran8/Vrooli/AGENTS.md)
- [x] Expanded [docs/repo-contract.md](/home/matthalloran8/Vrooli/docs/repo-contract.md) with preferred consumption order and grandfathered-debt rules
- [x] Expanded [docs/CONTRIBUTING.md](/home/matthalloran8/Vrooli/docs/CONTRIBUTING.md) with repo-contract adoption and validation guidance
- [x] Updated [cross-platform-readiness SKILL.md](/home/matthalloran8/Vrooli/scenarios/prompt-manager/store/skills/packs/core/cross-platform-readiness/SKILL.md) to stop teaching direct monorepo path joins
- [x] Added `adoption_rules_alignment` to [internal/repocontractcheck/checks.go](/home/matthalloran8/Vrooli/internal/repocontractcheck/checks.go)
- [x] Added failure-mode tests in [checks_test.go](/home/matthalloran8/Vrooli/internal/repocontractcheck/checks_test.go)

## Post-Phase-6 Burn-Down

Recently migrated debt:

- [x] `app-monitor/api/services/app_utils.go`
- [x] `app-monitor/api/handlers/system.go`
- [x] `prd-control-tower/api/main.go`
- [x] `scenario-stack-governor/api/repo_root.go`
- [x] `system-monitor/api/internal/services/paths.go`

Still worth follow-up:

- [ ] `prd-control-tower/cli/cmd_prd.go`
- [ ] `app-monitor/api/handlers/lighthouse.go`
- [ ] `secrets-manager` repo-root and manifest-resolution helpers
- [ ] `deployment-manager` monorepo fallback helpers
- [ ] `scenario-to-desktop` local root-detection portability helpers
- [ ] `vrooli-autoheal` monorepo fallback helpers

## Validation

- [x] `make validate-repo-contract`
- [x] `vrooli contract validate`
- [x] `go test ./internal/repocontractcheck ./internal/repocontract`

Focused consumer validation completed for the post-Phase-6 burn-down slice:

- [x] `cd scenarios/app-monitor/api && go test ./...`
- [x] `cd scenarios/prd-control-tower/api && go test ./...`
- [x] `cd scenarios/system-monitor/api && go test ./...`
- [x] `cd scenarios/scenario-stack-governor/api && go test ./...`
