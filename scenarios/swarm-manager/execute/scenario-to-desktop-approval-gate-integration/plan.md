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
- Wait/resume behavior when gate is blocked (poll with exponential backoff, configurable timeout)
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
- `api/pipeline/orchestrator.go:537-749` — runPipelineAsync: core execution loop, stage sequencing
- `api/pipeline/orchestrator.go:560-574` — Provenance capture and propagation
- `api/pipeline/stage_deploy.go:77-205` — Deploy stage Execute(): validates config, resolves target, uploads artifacts
- `api/pipeline/stage_deploy.go:84-89` — DeployConfig nil check (gate check inserts after this)
- `api/pipeline/provenance.go:13-66` — BuildProvenance struct with GitCommitHash
- `api/pipeline/types.go:25-33` — Status constants (Idle, Pending, Running, Completed, Failed, Cancelled, Skipped)
- `api/pipeline/types.go:37-100` — Pipeline state machine (Created → Initializing → QueueingStage → ExecutingStage → ProcessingResult → Completed)
- `api/pipeline/types.go:227-243` — DeployConfig struct (TargetName, ScenarioName, RemoteProfile, AppKey, UpdateURL)
- `api/pipeline/stage_helpers.go:243-299` — Poller[T] generic for async polling with timeout
- `api/deploy/targets.go:15-20` — DeployTarget struct (Label, ScenarioName, RemoteProfile)
- `api/deploy/targets.go:28-131` — TargetRepository: load/save deploy-targets.json with schema versioning
- `api/deploy/lpbs_client.go:164-217` — LPBS artifact upload orchestration

### deployment-manager approval API (contract)
- `POST /api/v1/profiles/{id}/approvals` — Create pending approval (git_commit_hash, platform)
- `GET /api/v1/profiles/{id}/release-gate?commit={hash}` — Check gate (returns ready bool + per-platform status)
- `GET /api/v1/profiles/{id}/approvals?commit={hash}` — List approvals
- `POST /api/v1/approvals/{id}/decide` — Approve/reject (decision, reviewer, notes)

### Existing discovery pattern
- `api/pipeline/manifest_generator.go:78` — `discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")` already resolves deployment-manager URL
- Import: `github.com/vrooli/api-core/discovery`

### Existing test patterns
- `api/pipeline/orchestrator_test.go:12-72` — mockTimeProvider, mockStage with ExecuteCh, ShouldFail/ShouldSkip
- `api/pipeline/stage_deploy_test.go:18-72` — testDeployFactory(), newTestDeployServer() with mock LPBS endpoints
- `api/pipeline/stage_helpers.go:55-99` — newStageResult(), failStage(), completeStage(), skipStage()

## 6. Target End State

1. Pipeline deploy stage auto-creates pending approval records on deployment-manager (one per built platform) before checking the gate
2. Pipeline deploy stage checks release gate before proceeding to artifact upload — blocks with clear status if not ready
3. Poll with exponential backoff (15s initial, 2x multiplier, 2m cap) and configurable timeout (default 30m); pipeline fails if the gate is not satisfied within timeout
4. CLI: `pipeline gate <pipeline-id>` shows current gate status; `pipeline resume <pipeline-id>` resumes a gate-blocked pipeline
5. UI: Deploy section shows gate status badge (Ready/Blocked/Waiting) with per-platform breakdown
6. All new code uses deployment-manager as the single source of truth — no local approval state
7. Pipeline fails if deployment-manager is unreachable (strict mode — approvals are a governance requirement)

## 7. Implementation Strategy

### Phase 1: Deployment-Manager API Client

**New file: `api/deploy/dm_client.go`**

Create a typed HTTP client for deployment-manager approval operations:
- `DMClient` struct with baseURL and http.Client
- Constructor: `NewDMClient(ctx context.Context)` — resolves URL via `discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")`
- Methods:
  - `CreateApproval(ctx, profileID, commitHash, platform string) (*Approval, error)` — POST /profiles/{id}/approvals
  - `CheckReleaseGate(ctx, profileID, commitHash string) (*ReleaseGateStatus, error)` — GET /profiles/{id}/release-gate?commit={hash}
  - `ListApprovals(ctx, profileID, commitHash string) ([]Approval, error)` — GET /profiles/{id}/approvals?commit={hash}
- Response types: `Approval`, `ReleaseGateStatus` (with Ready bool, per-platform breakdown)
- All methods return structured errors (connection refused, 4xx, 5xx) for the pipeline to handle

### Phase 2: DeployTarget Schema Update

**Modify: `api/deploy/targets.go`**

- Add `DeploymentManagerProfileID string \`json:"deployment_manager_profile_id,omitempty"\`` to DeployTarget struct (line ~18)
- No schema version bump — field is optional and backward-compatible (round 2 d3, pending)
- When empty, approval gating is skipped for that target

**Modify: `api/pipeline/types.go`**

- Add to DeployConfig (line ~243):
  - `DeploymentManagerProfileID string` — from deploy target or explicit config
  - `GateTimeout time.Duration` — max wait for gate (default 30m)
  - `GatePollInterval time.Duration` — initial poll interval (default 15s)

### Phase 3: Pipeline Integration — Deploy Stage Gate Check

**Modify: `api/pipeline/stage_deploy.go`**

Insert gate logic at the top of Execute(), after DeployConfig validation (line ~89):

1. **Create approvals:** For each platform in build results, call `dm_client.CreateApproval(profileID, commitHash, platform)`
2. **Check gate:** Call `dm_client.CheckReleaseGate(profileID, commitHash)`
3. **If gate ready:** Proceed to artifact upload
4. **If gate blocked:** Use `Poller[ReleaseGateStatus]` with exponential backoff (15s → 30s → 60s → 2m cap) and GateTimeout
   - Signal gate-blocked state to the pipeline (mechanism TBD — round 2 d1)
   - On timeout: fail the stage with clear error listing which platforms are still blocked
5. **If deployment-manager unreachable:** Fail the stage immediately (round 1 d5=B — strict mode)
6. **If no profile ID configured:** Skip approval gating entirely (log info message)

### Phase 4: CLI Integration

**Modify: `cli/pipeline/commands.go`**

- Add `--deployment-profile` flag to Run command (line ~140 area) — maps to DeployConfig.DeploymentManagerProfileID
- Add `--gate-timeout` flag (default "30m")
- Add `--gate-poll-interval` flag (default "15s")
- Add `gate` subcommand: fetches pipeline status and shows gate state (ready/blocked/waiting with per-platform breakdown)
- Existing `resume` command should work for gate-blocked pipelines (verify and test)

### Phase 5: UI Integration

**Modify: `ui/src/components/sections/deploy/DeploySection.tsx`**

- Add gate status section above the existing deploy content
- States: "Checking gate..." (spinner), "Gate passed" (green badge), "Waiting for approval" (amber badge with per-platform list), "Gate failed" (red badge)
- Per-platform breakdown: show each platform with its approval status (pending/approved/rejected)
- When waiting: show elapsed time and timeout

### Phase 6: Tests

**New file: `api/deploy/dm_client_test.go`**
- Unit tests with httptest.NewServer mocking deployment-manager responses
- Test scenarios: successful approval creation, gate ready, gate blocked, timeout, connection refused, 4xx/5xx responses
- Test approval creation for multiple platforms

**Modify: `api/pipeline/stage_deploy_test.go`**
- Add test: gate check passes → deploy proceeds normally
- Add test: gate blocked → poll → gate clears → deploy proceeds
- Add test: gate blocked → timeout → stage fails with clear error
- Add test: deployment-manager unreachable → stage fails immediately
- Add test: no profile ID configured → gate check skipped, deploy proceeds
- Use newTestDeployServer() pattern to add deployment-manager mock endpoints

**CLI tests:**
- Test --deployment-profile flag passes through to DeployConfig
- Test `gate` subcommand output for ready/blocked/waiting states

**UI tests:**
- Test gate badge rendering for all states (checking, passed, waiting, failed)
- Test per-platform approval breakdown display

## 8. Contract Decisions

### Settled (Round 1)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Profile ID resolution | Add `deployment_manager_profile_id` to DeployTarget | Keeps all deploy config in one place; deploy-target doctor can validate |
| Gate check placement | First step inside Deploy stage | No new stage; consistent with deploy readiness concept |
| Wait behavior | Poll with exponential backoff + configurable timeout | Simple, uses existing Poller[T], no new infrastructure |
| Approval creation timing | Automatically after build (at deploy stage start) | Natural point where artifacts exist and need approval |
| Degradation mode | Strict — fail if deployment-manager unreachable | Approvals are a governance requirement |

### Pending (Round 2)

| Decision | Status |
|----------|--------|
| Pipeline state for gate-blocked | Awaiting selection |
| Approval creation mechanism | Awaiting selection |
| deploy-targets.json schema versioning | Awaiting selection |
| Gate polling defaults | Awaiting selection |

## 9. Testing Plan

### Unit Tests

| Component | Test | Expected |
|-----------|------|----------|
| dm_client.go | CreateApproval success | Returns Approval with ID, status=pending |
| dm_client.go | CreateApproval 409 (duplicate) | Idempotent — returns existing approval |
| dm_client.go | CheckReleaseGate ready | Returns ReleaseGateStatus{Ready: true} |
| dm_client.go | CheckReleaseGate blocked | Returns ReleaseGateStatus{Ready: false, Platforms: [...]} |
| dm_client.go | Connection refused | Returns typed error, not panic |
| dm_client.go | 500 response | Returns typed error with status code |
| stage_deploy.go | Gate ready → deploy | Approvals created, gate checked, artifacts uploaded |
| stage_deploy.go | Gate blocked → poll → clear | Polls with backoff, resumes when gate clears |
| stage_deploy.go | Gate blocked → timeout | Fails with error listing blocked platforms |
| stage_deploy.go | DM unreachable | Stage fails immediately (strict mode) |
| stage_deploy.go | No profile ID | Gate check skipped, deploy proceeds normally |

### Integration Tests

| Scenario | Verification |
|----------|-------------|
| Full pipeline with gate pass | Build → create approvals → gate check → deploy succeeds |
| Full pipeline with gate block + resume | Build → create approvals → gate blocked → manual approval → resume → deploy |

### CLI Tests

| Command | Verification |
|---------|-------------|
| `pipeline run --deployment-profile X` | Profile ID passed to DeployConfig |
| `pipeline gate <id>` | Shows gate status with per-platform breakdown |

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
- [ ] CLI shows gate status with per-platform breakdown
- [ ] UI shows gate status badge
- [ ] Strict mode: pipeline fails when deployment-manager is unreachable

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| deployment-manager unreachable during pipeline | Medium | High | Strict fail — pipeline stops, user investigates. Clear error message with DM URL and health check suggestion |
| Profile ID not configured on deploy target | Medium | Low | Skip gate check, log info. Deploy-target doctor warns about missing profile ID |
| Gate polling creates excessive load | Low | Medium | Exponential backoff (15s→2m cap), max 30m timeout. ~25 requests over 30m worst case |
| Stale approvals after code change mid-pipeline | Low | Medium | Provenance captures exact commit; approvals are tied to commit hash |
| Pipeline blocked indefinitely | Low | High | Configurable timeout with clear error message listing blocked platforms |
| CreateApproval called for already-existing approval | Low | Low | deployment-manager should handle 409 idempotently; dm_client treats 409 as success |

## 12. Non-goals / Prohibited Patterns

- Do NOT store approval state locally — deployment-manager is the single source of truth
- Do NOT duplicate gate-checking logic — call deployment-manager's release-gate endpoint
- Do NOT modify deployment-manager API endpoints
- Do NOT add approval decide/approve/reject functionality to scenario-to-desktop (that's deployment-manager's domain)
- Do NOT implement webhook/push notifications (use explicit poll-with-timeout)
- Do NOT add a new pipeline stage for gate check (settled: gate check is inside Deploy stage)
- Do NOT bump deploy-targets.json schema version for the optional profile ID field (pending round 2 d3)

## 13. Definition of Done

1. `api/deploy/dm_client.go` exists with typed methods for CreateApproval, CheckReleaseGate, ListApprovals
2. `api/deploy/dm_client_test.go` has unit tests for all methods including error scenarios
3. DeployTarget has `deployment_manager_profile_id` optional field
4. DeployConfig has `GateTimeout` and `GatePollInterval` fields with defaults
5. Deploy stage creates pending approvals and checks gate before artifact upload
6. Deploy stage polls with exponential backoff when gate is blocked, fails on timeout
7. Deploy stage fails immediately when deployment-manager is unreachable
8. CLI exposes `--deployment-profile`, `--gate-timeout`, `--gate-poll-interval` flags and `gate` subcommand
9. UI DeploySection shows gate status badge with per-platform breakdown
10. All new code has automated tests; `go build` and `go test` pass for api/ and cli/; UI builds and tests pass
