# Plan: Fix swarm-manager execution tracking not recording completed runs

## Purpose

Completed agent-manager runs are invisible to swarm-manager, blocking the review workflow. Four bugs in the execution polling pipeline prevent status transitions from reaching terminal states. This plan fixes all four bugs and adds regression tests so the polling pipeline reliably records completed runs.

## Required Reading

- `prompt-manager skill read scientific-debugging` — Hypothesis-driven root cause analysis
- `prompt-manager skill read api-steer` — API design patterns for inter-service communication
- `prompt-manager skill read seam-discovery-and-enforcement` — Testability boundaries

## Current Technical Context

### Architecture

```
swarm-manager (QueueBacklog / startLocked)
  -> agentmanager.AgentService.SpawnBacklog()
    -> POST /api/v1/tasks (create task)
    -> POST /api/v1/runs  (create run, get run.Id)
  <- stores run.Id in execution Record (.RunID)
  <- saves Record to execution-runs.json with Status=StatusStarting

[every 2 seconds, background goroutine]
swarm-manager execution.Handler.StartBackgroundWorker()
  -> ProcessActiveExecutions()
    -> refreshRunningLocked()
      -> agentactivity.Service.GetRunState(runID)
        -> GET /api/v1/runs/{runID} on agent-manager
      -> mapRunStatus(run.Status) -> nextStatus
      -> if changed: save to execution-runs.json, update backlog spec.json
      -> if completed+finalization-eligible: set StatusValidating, spawn processFinalization

agent-manager (RunExecutor.Execute)
  -> handleResult() -> runs.Update() -> SQLite
  [no callback to swarm-manager]
```

### Key Files

| File | Role |
|------|------|
| `scenarios/swarm-manager/api/internal/execution/service.go` | Service constructor, type assertions for differ/inspector/continuer/stopper |
| `scenarios/swarm-manager/api/internal/execution/polling.go` | Core polling loop, `mapRunStatus`, `refreshRunningLocked` |
| `scenarios/swarm-manager/api/internal/execution/handler.go` | Background polling worker (2s ticker) |
| `scenarios/swarm-manager/api/internal/execution/finalization_scope.go` | Sandbox diff fetching (broken due to nil differ) |
| `scenarios/swarm-manager/api/internal/execution/store.go` | Persists execution records to `execution-runs.json` |
| `scenarios/swarm-manager/api/internal/execution/backlog_bridge.go` | Updates backlog item status on completion |
| `scenarios/swarm-manager/api/routes_execution.go` | Wires AgentService into ServiceConfig |
| `scenarios/swarm-manager/api/internal/agentactivity/` | Activity-tracking wrapper around agentmanager.AgentService |
| `scenarios/swarm-manager/api/main.go` | Creates agentmanager.AgentService (line 84), wraps in agentactivity.Service (line 208) |

## Problem Statement — Root Cause Analysis

Four bugs in the execution polling pipeline prevent completed runs from being recorded:

### Bug 1 (Critical): `agentactivity.Service` does not implement `RunDiffer`

- In `service.go` lines 213-224, the `differ` field is set via type assertion: `if differ, ok := cfg.AgentService.(RunDiffer); ok`
- `routes_execution.go` line 29 passes `s.requireTrackedAgentService()` which returns `*agentactivity.Service`
- `agentactivity.Service` wraps `agentmanager.AgentService` but does NOT forward `GetRunDiff`
- Result: `service.differ` is always `nil`
- In `finalization_scope.go` line 20, the `if s.differ != nil` check always fails, so sandbox diffs are never fetched
- Without diffs, finalization falls back to `acceptance_allow` globs; if empty, finalization is skipped entirely

### Bug 2 (Critical): `mapRunStatus` treats unknown statuses as `StatusRunning`

- In `polling.go` lines 207-227, the default case returns `StatusRunning`
- This includes `"unspecified"` from `normalizeRunStatus` when the proto enum is zero/default
- Result: runs with unrecognized statuses are polled forever and never reach a terminal state

### Bug 3 (Moderate): No staleness detection in execution poller

- `polling.go` line 116-119: `if err != nil { continue }` — all errors silently ignored
- If agent-manager restarts or loses the run record, swarm-manager polls forever
- No timeout for how long a run can stay in `StatusStarting` or `StatusRunning`

### Bug 4 (Low): `SelfScenarioName` fallback is fragile

- `service.go` lines 190-193: uses `filepath.Base(rootDir)` when `cfg.SelfScenarioName` is empty
- `routes_execution.go` does not set `SelfScenarioName` in `ServiceConfig`
- If the directory isn't named `swarm-manager`, the self-restart skip logic fails

## Scope

**In scope:**
- Fix `agentactivity.Service` to forward `GetRunDiff` to the inner client (Bug 1)
- Fix `mapRunStatus` default case to transition unknown statuses to `StatusFailed` after a grace period (Bug 2)
- Add staleness detection with consecutive-error counter and max-age timeout (Bug 3)
- Set `SelfScenarioName` explicitly in `routes_execution.go` (Bug 4)
- Add per-run ephemeral tracking state via in-memory `map[runID]*runTracker` (not persisted)
- Unit tests for all fixes

**Out of scope:**
- Changing the polling model to push/webhook
- Modifying agent-manager's internal run lifecycle
- UI changes for execution display
- Backward-compatibility shims for old `execution-runs.json` format
- Changes to files outside `acceptance_allow` patterns

## Target End State

After this fix:
1. `agentactivity.Service` forwards `GetRunDiff` to the inner client, so `service.differ` is non-nil
2. `mapRunStatus` returns a terminal error status for unknown status strings after 5 consecutive unknown polls (~10 seconds)
3. The polling loop tracks consecutive errors and run age via an in-memory `map[runID]*runTracker`, marking runs as failed after 30 consecutive errors (~60 seconds) or 30 minutes max-age
4. `SelfScenarioName` is explicitly set in `routes_execution.go`
5. All fixes have unit tests that would catch regressions

## Contract Decisions

| Decision | Outcome | Source |
|----------|---------|--------|
| RunDiffer wiring fix | Add `GetRunDiff` forwarding method to `agentactivity.Service` | Workshop R1 d1 → A |
| Staleness detection strategy | Configurable timeout with consecutive-error counter | Workshop R1 d2 → A (user note: default > 20s) |
| Unknown status handling | Log warning on first unknown status, keep StatusRunning; after grace period transition to StatusFailed | Workshop R1 d3 → A |
| Staleness threshold defaults | 30 consecutive errors (~60s) + 30 min max-age, both configurable via ServiceConfig | Workshop R2 d1 → A |
| Per-run tracking state storage | In-memory `map[runID]*runTracker` alongside the store; not persisted; created/deleted in sync with execution records | Workshop R2 d2 → A |
| Unknown-status grace period | 5 consecutive unknown polls (~10 seconds) before transitioning to StatusFailed | Workshop R2 d3 → A |

## Implementation Strategy

This is a greenfield fix — no backward-compatibility shims needed.

### Phase 1: Fix RunDiffer wiring (Bug 1)

1. Add a `GetRunDiff(ctx context.Context, runID string) (agentmanager.RunDiff, error)` method to `agentactivity.Service` that delegates to the inner `agentmanager.AgentService`
2. Verify with a unit test that `NewService` produces a non-nil `differ` when `AgentService` is an `agentactivity.Service` wrapping a `RunDiffer`-implementing client

### Phase 2: Fix mapRunStatus default case (Bug 2)

1. Add a `ConsecutiveUnknown int` field to the per-run `runTracker` (in-memory `map[runID]*runTracker`)
2. On first unknown status: log a warning at WARN level, keep `StatusRunning`, increment `ConsecutiveUnknown`
3. After 5 consecutive polls returning an unknown status (~10 seconds), return `StatusFailed`
4. On any known status: reset `ConsecutiveUnknown` to 0
5. Add explicit mappings for all known agent-manager run statuses to prevent false unknowns
6. Unit test: verify unknown status → warning on first call, failure after 5 consecutive unknowns

### Phase 3: Add staleness detection (Bug 3)

1. Add fields to the `runTracker` struct: `ConsecutiveErrors int`, `FirstSeen time.Time`
2. On `GetRunState` error: increment `ConsecutiveErrors`, log at WARN level
3. After 30 consecutive errors (~60 seconds), mark run as `StatusFailed`
4. Add a max-age timeout: if `time.Since(FirstSeen)` exceeds 30 minutes, mark as `StatusFailed`
5. On successful poll: reset `ConsecutiveErrors` to 0
6. Both thresholds configurable via `ServiceConfig` fields (`MaxConsecutiveErrors int`, `MaxRunAge time.Duration`)
7. Unit test: verify error accumulation triggers failure; verify max-age triggers failure

### Phase 4: Fix SelfScenarioName (Bug 4)

1. Set `SelfScenarioName: "swarm-manager"` explicitly in `routes_execution.go` ServiceConfig
2. This is a one-line fix, no separate test needed (covered by existing self-restart skip tests)

### Phase 5: Regression tests

1. `mapRunStatus` test table covering all known statuses + unknown + empty
2. `NewService` test verifying `differ` non-nil with proper wiring
3. Polling loop test with mock inspector returning errors → 30 consecutive errors → failure
4. Polling loop test with mock inspector returning unknown status → 5 consecutive unknowns → failure
5. Staleness max-age test with time manipulation
6. Error-counter reset test: successful poll after errors resets counter

## Non-goals / Prohibited Patterns

- Do NOT change the polling model to push/webhook — that's a separate initiative
- Do NOT modify agent-manager's internal run lifecycle
- Do NOT add UI changes for execution display
- Do NOT add backward-compatibility shims for old execution-runs.json format
- Do NOT persist per-run tracking state (ConsecutiveErrors, ConsecutiveUnknown) — these are ephemeral by design

## Testing Plan

| Test | Type | What It Validates |
|------|------|-------------------|
| `TestMapRunStatus_AllKnown` | Unit | Every known agent-manager status maps correctly |
| `TestMapRunStatus_Unknown_GracePeriod` | Unit | Unknown status returns Running initially, Failed after 5 consecutive unknowns |
| `TestNewService_DifferWired` | Unit | agentactivity.Service with RunDiffer inner → service.differ non-nil |
| `TestPolling_ConsecutiveErrors` | Unit | 30 consecutive GetRunState errors → run marked Failed |
| `TestPolling_MaxAge` | Unit | Run exceeding 30 min max-age timeout → marked Failed |
| `TestPolling_ErrorReset` | Unit | Successful poll resets consecutive error counter |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Changing mapRunStatus default may cause false failures for legitimate new statuses | Medium | Grace period (5 polls / ~10s) before failing; log warning on first unknown |
| Staleness timeout too aggressive could kill slow-running agents | Medium | Default 30 min max-age; configurable via ServiceConfig; consecutive-error threshold (30 = ~60s) requires sustained failure |
| Differ fix may change finalization behavior for existing runs | Low | Only applies to future runs; existing completed runs unaffected |
| Consecutive-error counter state lost on restart | Low | Acceptable by design — restart resets counters, runs resume polling fresh |
| In-memory runTracker map must stay in sync with store | Low | Create entries when records are created, delete on terminal state; single goroutine owns both |

## Acceptance Criteria / Definition of Done

- [ ] Completed runs in agent-manager are correctly recorded in swarm-manager's execution store
- [ ] Backlog items transition to their post-execution status when runs complete
- [ ] Unknown run statuses do not cause infinite polling (fail after ~10s of unknown)
- [ ] Stale/lost runs are detected and marked as failed after ~60s of consecutive errors or 30 min max-age
- [ ] Per-run tracking state is ephemeral (in-memory map, not persisted)
- [ ] All 6 test cases pass
- [ ] `go build ./...` and `go test ./...` pass for swarm-manager
- [ ] `vrooli scenario restart swarm-manager` succeeds with clean startup

## Rollout / Validation Checklist

1. Apply all code changes (Phases 1-4)
2. Run `cd scenarios/swarm-manager/api && go build ./...` — must pass
3. Run `cd scenarios/swarm-manager/api && go test ./... -timeout 300s` — must pass
4. `vrooli scenario restart swarm-manager`
5. Queue a test backlog item, let agent-manager complete it, verify execution appears in swarm-manager
