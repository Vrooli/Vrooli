# Implementation Plan: Integrate Scenario-to-Desktop with Canonical Approval Gates

## 1. Purpose

Make scenario-to-desktop a first-class participant in the deployment-manager approval-gating flow. After building desktop artifacts, the pipeline should create pending approvals, check gate status before deploying, surface blocked conditions in CLI/UI, and support wait/resume behavior — all without duplicating approval state or business rules.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer seam-discovery-and-enforcement implementation-plan-authoring
```

**Research dependency:** `research/desktop-release-control-plane-audit` (completed) — system contracts, interface boundaries, and gap analysis for the 4-system desktop release pipeline.

**Execute dependency:** `execute/deployment-manager-approval-gate-surfaces` (completed) — deployment-manager now exposes all 7 approval endpoints via CLI and UI.

**Key findings from research:**
- deployment-manager owns canonical approval state in `deployment_approvals` table (Finding 1)
- Release gate: `GET /profiles/{id}/release-gate?commit={hash}` returns ready/blocked with per-platform status
- Orchestrator blocks deploy with HTTP 412 when gate is not satisfied
- Build provenance (git commit hash) is already captured by scenario-to-desktop and passed to LPBS
- scenario-to-desktop talks to LPBS via `lpbs_client.go`; deployment-manager talks to scenario-to-desktop via `desktop_client.go`

## 3. Problem Statement

scenario-to-desktop builds and publishes desktop artifacts but has no awareness of the approval-gating flow:
- After a build completes, no pending approvals are created on deployment-manager
- The pipeline has no way to check if the release gate is satisfied before deploying
- CLI and UI don't show gate status or blocked conditions
- There's no wait/resume pattern — if the gate is blocked, the pipeline either fails or proceeds blindly
- Build provenance is captured but not used to create approval records

## 4. Scope

**In scope:**
- New deployment-manager API client in scenario-to-desktop for approval operations
- Pipeline integration: create pending approvals after build, check gate before deploy
- Wait/resume behavior when gate is blocked (configurable timeout)
- CLI commands for approval status and gate check
- UI gate status display in the deploy section
- Automated tests for new integration points

**Out of scope:**
- Modifying deployment-manager approval endpoints (already complete)
- Modifying LPBS endpoints or schema
- Approval-to-deploy automation (separate backlog item)
- Rollback API (separate backlog item)
- Post-publish version sync callback (separate backlog item)
- Visual validation integration (separate workflow)

## 5. Current Technical Context

### scenario-to-desktop pipeline (key files)
- `api/pipeline/orchestrator.go:560-574` — Provenance capture and propagation
- `api/pipeline/stage_deploy.go:77-205` — Deploy stage: resolves target, uploads artifacts to LPBS
- `api/pipeline/provenance.go:13-66` — BuildProvenance struct with GitCommitHash
- `api/pipeline/types.go:284-355` — Status struct with Provenance field
- `api/deploy/lpbs_client.go:164-217` — LPBS artifact upload orchestration

### deployment-manager approval API (contract)
- `POST /api/v1/profiles/{id}/approvals` — Create pending approval (git_commit_hash, platform)
- `GET /api/v1/profiles/{id}/release-gate?commit={hash}` — Check gate (returns ready bool + per-platform status)
- `GET /api/v1/profiles/{id}/approvals?commit={hash}` — List approvals
- `POST /api/v1/approvals/{id}/decide` — Approve/reject (decision, reviewer, notes)

### Existing deployment-manager client
- `scenarios/deployment-manager/api/deployments/desktop_client.go` — deployment-manager→scenario-to-desktop calls (reverse direction)
- `api/pipeline/manifest_generator.go:71-82` — scenario-to-desktop already resolves deployment-manager URL via discovery

## 6. Target End State

1. Pipeline `build` stage completion auto-creates pending approval records on deployment-manager (one per built platform)
2. Pipeline `deploy` stage checks release gate before proceeding — blocks with clear status if not ready
3. Configurable wait/resume: pipeline can poll gate status with timeout, resuming deploy when gate clears
4. CLI: `pipeline gate <pipeline-id>` shows current gate status; `pipeline resume <pipeline-id>` resumes a gate-blocked pipeline
5. UI: Deploy section shows gate status badge (Ready/Blocked/Waiting) with per-platform breakdown
6. All new code uses deployment-manager as the single source of truth — no local approval state

## 7. Implementation Strategy

<!-- TBD — pending decisions on gate check placement, wait behavior, and profile ID resolution -->

### Phase 1: Deployment-Manager API Client
- Create `api/deploy/dm_client.go` with typed methods for approval operations
- Use `discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")` for URL resolution (same pattern as manifest_generator.go)
- Methods: CreateApproval, CheckReleaseGate, ListApprovals

### Phase 2: Pipeline Integration — Create Approvals After Build
- After build stage completes, create pending approvals on deployment-manager for each successfully built platform
- Requires: deployment profile ID, git commit hash, platform list from build results
- Profile ID resolution strategy: TBD (decision d1)

### Phase 3: Pipeline Integration — Gate Check Before Deploy
- Gate check placement: TBD (decision d2)
- Wait/resume behavior: TBD (decision d3)
- Deploy stage checks gate; if blocked, pipeline enters "waiting" state

### Phase 4: CLI Integration
- Add gate status and resume commands to pipeline CLI group
- Human-friendly output showing per-platform approval status

### Phase 5: UI Integration
- Gate status badge in DeploySection component
- Per-platform approval breakdown
- Visual indication of waiting state

### Phase 6: Tests
- Unit tests for dm_client.go with httptest mocks
- Integration tests for pipeline gate check flow
- CLI command tests
- UI component tests

## 8. Contract Decisions

<!-- Accumulated from workshop rounds -->

## 9. Testing Plan

<!-- TBD — pending approach decisions -->

- dm_client.go: unit tests with httptest.NewServer mocking deployment-manager responses
- Pipeline gate check: test blocked (412-equivalent), ready, and timeout scenarios
- CLI gate command: test output formatting for ready/blocked/waiting states
- UI DeploySection: test gate badge rendering for all states

## 10. Rollout / Validation Checklist

- [ ] `cd scenarios/scenario-to-desktop/api && go build ./...`
- [ ] `cd scenarios/scenario-to-desktop/api && go test ./... -timeout 300s`
- [ ] `cd scenarios/scenario-to-desktop/cli && go build ./...`
- [ ] `cd scenarios/scenario-to-desktop/cli && go test ./... -timeout 300s`
- [ ] `cd scenarios/scenario-to-desktop/ui && npm run build`
- [ ] `cd scenarios/scenario-to-desktop/ui && npm test`
- [ ] Pipeline creates pending approvals after build
- [ ] Pipeline checks gate before deploy and blocks when not ready
- [ ] Wait/resume works with configurable timeout
- [ ] CLI shows gate status
- [ ] UI shows gate status badge

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| deployment-manager unreachable during pipeline | Medium | High | Graceful degradation: warn but don't block pipeline if approval service is unavailable (configurable) |
| Profile ID unknown at pipeline time | Medium | High | Resolve via deploy target config or explicit pipeline parameter |
| Gate polling creates excessive load | Low | Medium | Exponential backoff with configurable max interval |
| Stale approvals after code change mid-pipeline | Low | Medium | Provenance captures exact commit; deployment-manager handles staleness |
| Pipeline blocked indefinitely | Low | High | Configurable timeout with clear error message and resume capability |

## 12. Non-goals / Prohibited Patterns

- Do NOT store approval state locally — deployment-manager is the single source of truth
- Do NOT duplicate gate-checking logic — call deployment-manager's release-gate endpoint
- Do NOT modify deployment-manager API endpoints
- Do NOT add approval decide/approve/reject functionality to scenario-to-desktop (that's deployment-manager's domain)
- Do NOT implement webhook/polling for real-time updates (use explicit gate check + poll-with-timeout)

## 13. Definition of Done

1. dm_client.go exists with typed methods for CreateApproval, CheckReleaseGate, ListApprovals
2. Pipeline creates pending approvals after successful build stage
3. Pipeline checks release gate before deploy stage and blocks when not ready
4. Wait/resume behavior works with configurable timeout
5. CLI exposes gate status and resume commands
6. UI DeploySection shows gate status badge with per-platform breakdown
7. All new code has automated tests
8. `go build` and `go test` pass for api/ and cli/
9. UI builds and tests pass
