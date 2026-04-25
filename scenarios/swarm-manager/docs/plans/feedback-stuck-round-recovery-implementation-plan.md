# Feedback Round Stuck-State Recovery — Implementation Plan

## 1. Purpose

Make the **initiative feedback** feature in `scenarios/swarm-manager` reliable in the face of agent-run failures. Today, a feedback round can wedge in `agent_thinking` indefinitely with no user-facing recovery path: the spawn succeeded, the agent later crashed (or never reported back), and the round + initiative lock stay held forever. This plan adds a cancel/recover pathway, automatic stuck-state detection, and resilient polling — so users can always escape and the system can self-heal.

This is a **greenfield** change. We are not preserving any compatibility shim for the current stuck rounds; existing wedged rounds will be migrated by the recovery sweep on startup.

## 2. Required Reading

Run before implementation:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Files the implementer must read first (with line anchors that ground every change in this plan):

- `scenarios/swarm-manager/api/internal/feedback/types.go` — round states & shape (lines 22–50, 124–151)
- `scenarios/swarm-manager/api/internal/feedback/service.go` — orchestration (lines 111–166 wiring; 193–391 StartRound; 540–572 Continue; 575–618 EnsurePolledTurn; 646–695 RecordAgentTurn)
- `scenarios/swarm-manager/api/internal/feedback/handler.go` — HTTP routing (lines 51–63 routes; 187–210 Get; 323–358 Dismiss)
- `scenarios/swarm-manager/api/internal/initiativelock/lock.go` — lock state machine (lines 100–185)
- `scenarios/swarm-manager/api/routes_feedback.go` — agent-manager spawner adapter & DI (lines 40–241)
- `scenarios/swarm-manager/ui/src/components/initiative/feedback-round-card.tsx` — UI of a single round; current `isActive` branch shows only a spinner (lines 130, 215–220)
- `scenarios/swarm-manager/ui/src/components/initiative/feedback-panel.tsx` — list/poll shell (refetchInterval at 52–65)
- `scenarios/swarm-manager/ui/src/services/feedback-service.ts` — TS API client
- `scenarios/swarm-manager/cli/domains/initiatives/register.go` — CLI registration (lines 27–35; this is where new feedback-cancel command goes)

## 3. Problem Statement

Reproducer (current production state, "Command Center Foundation" initiative):

1. User submits feedback → `StartRound` saves round as `agent_thinking`, takes the lock, calls `SpawnInitiativeFeedback` and persists the returned `RunID` (`service.go:193–391`, `routes_feedback.go:189–241`).
2. Agent-manager run crashes / dies / disappears.
3. UI polls `GET /api/v1/initiatives/{name}/feedback/{round}`; `EnsurePolledTurn` (`service.go:586–600`) calls `poller.GetRunState(runID)`; the poller returns an error or non-terminal status; the function logs `slog.Warn` and **silently returns the unchanged round**.
4. No timeout, no cancel endpoint, no UI button when `isActive`. Round and lock stay held forever.

Three independent gaps cause this:

- **G1 — No cancel surface**: `Handler.RegisterRoutes` exposes Start/List/Get/Continue/Decide/Dismiss/AgentTurn/GetAttachment/LockStatus only. There is no endpoint that means "stop this run and free this round." `Dismiss` exists but is wrapped in `Decide(kind=dismiss)` which has no special handling for `agent_thinking` (it does not call `StopRun` or release the lock explicitly — and the UI hides Dismiss while `isActive`).
- **G2 — No stuck-state detection**: `EnsurePolledTurn` swallows poller errors. There is no "run-not-found ⇒ terminal-failure" mapping. There is no time-based sweep of `agent_thinking` rounds.
- **G3 — No UI recovery affordance**: `feedback-round-card.tsx:215–220` renders a spinner for `isActive` with no buttons. Users cannot dismiss, cancel, or even see why the agent appears stuck.

## 4. Scope

### In scope
- New `POST /api/v1/initiatives/{name}/feedback/{round}/cancel` endpoint that calls `RunCanceller.StopRun`, releases the lock, and lands the round in `dismissed` with rationale.
- Resilient `EnsurePolledTurn` that maps "run not found" / persistent-failure poller responses to a terminal-failure agent turn.
- Background **stuck-round sweeper**: a goroutine on the swarm-manager API that periodically walks `agent_thinking` rounds and force-recovers any whose run is gone or whose `updated_at` exceeds a configurable max age (default 30 min).
- New `last_poll_error` and `last_polled_at` round fields surfaced to the UI for visibility.
- UI: `feedback-round-card.tsx` `isActive` branch gains a **Cancel** button (red, secondary) and renders `last_poll_error` when present.
- New CLI command `swarm-manager initiatives feedback-cancel --name N --round R` mirroring the new endpoint (per `cli-steer`: every API endpoint has a CLI command).
- Automated tests covering: cancel happy-path, cancel while spawner unreachable, sweep dismisses old rounds, EnsurePolledTurn maps run-not-found to failure turn, UI renders + clicks the Cancel button.
- One-time recovery on startup: the existing `lock.SweepStale` plus a new `feedback.SweepStuckRounds(ctx)` invocation that closes orphan `agent_thinking` rounds whose lock no longer exists.

### Out of scope
- Reworking the `Decide` flow or the proposal-application pipeline.
- Changing how the agent-manager itself reports run status (we only consume what it gives us).
- Reworking research-type rounds (still scaffolded; this plan only touches `feedback` and `note` rounds).
- Adding a "retry the agent" button — different feature; mention in §11 as future work.
- Webhook-based agent-turn delivery from agent-manager. Polling stays the source of truth.

## 5. Current Technical Context

| Concern | File | Notes |
|---|---|---|
| Round states | `api/internal/feedback/types.go:22–50` | Terminal states: applied, rejected, dismissed |
| Service wiring | `api/internal/feedback/service.go:111–166` | `Service{spawner, poller, canceller}` already in struct |
| Spawn → persist RunID | `api/internal/feedback/service.go:299–391` | RunID written to round + lock |
| Poll-and-advance | `api/internal/feedback/service.go:586–600` | Logs and no-ops on poller error |
| Failure-mapping helper | `api/internal/feedback/service.go:622–636` | `isTerminalRunStatus` / `isFailureRunStatus` |
| Record agent turn | `api/internal/feedback/service.go:646–695` | Releases lock at line 693 |
| Override path (existing cancel ref) | `api/internal/feedback/service.go:417, 436` | Already calls `canceller.StopRun` |
| HTTP routes | `api/internal/feedback/handler.go:51–63` | Register the new `/cancel` route here |
| Dismiss handler | `api/internal/feedback/handler.go:323–358` | Pattern to mirror |
| Spawner adapter | `api/routes_feedback.go:189–241` | `SpawnInitiativeFeedback`/`ContinueRun` lives here; wire-in canceller already present |
| Lock | `api/internal/initiativelock/lock.go:100–185` | `Acquire`, `Release`, `SweepStale`, `Inspect` |
| UI active-state branch | `ui/src/components/initiative/feedback-round-card.tsx:215–220` | Insert Cancel button here |
| UI polling shell | `ui/src/components/initiative/feedback-panel.tsx:52–65` | `refetchInterval` already set |
| UI feedback service | `ui/src/services/feedback-service.ts` | Add `cancel()` method |
| CLI registration | `cli/domains/initiatives/register.go:27–35` | Add `feedback-cancel` next to `feedback-decide` |

The `RunCanceller` interface and an actual canceller implementation already exist (`service.go:65–67`, used in override at `service.go:417` & `436`), so cancel does not require new agent-manager wiring — only a new endpoint that calls into it.

## 6. Target End State

Behavioral guarantees after this plan ships:

1. **No round stays in `agent_thinking` longer than `FEEDBACK_STUCK_MAX_AGE` (default 30 min) without resolution.** The sweeper will either advance it (poller now reports terminal) or force-dismiss it with a clear rationale.
2. **Every `agent_thinking` round in the UI has a visible Cancel button** that succeeds even if the agent process is gone.
3. **`Dismiss` on an `agent_thinking` round** still works as a fallback: the existing `Decide(kind=dismiss)` path is upgraded to call `canceller.StopRun` and release the lock if the round was active. (No new endpoint needed for this — same shape.)
4. **Polling errors are surfaced**, not silent. `last_poll_error` is persisted on the round and rendered.
5. **All four current stuck rounds in production** (including the Command Center Foundation one) are auto-dismissed by the startup sweep on first deploy.
6. **Feature parity across surfaces**: API endpoint + TS service method + CLI command + UI button.

## 7. Implementation Strategy (Phased)

Phases are dependency-ordered. Each phase is independently testable and lands its own commit-able unit of work.

### Phase 1 — API: Resilient polling + new fields (no new routes)
- Add fields to `Round` in `types.go`: `LastPolledAt string`, `LastPollError string` (both omitempty).
- In `service.go:EnsurePolledTurn`:
  - Always set `LastPolledAt` and persist when polling occurs.
  - On poller error: set `LastPollError`, persist round, return.
  - When poller returns `not_found` / `unknown` / consistent-error for `> N` consecutive polls (track via `LastPolledAt` and a new `PollFailureCount int`), synthesize a failure-terminal status and call `RecordAgentTurn` with body `"agent run failed: run no longer reachable (<error>)"`.
  - Extend `isFailureRunStatus` to recognize `"not_found"` and `"missing"`.
- Update `service_test.go` and `ensure_polled_test.go` for the new branches.

### Phase 2 — API: Cancel endpoint + service method
- New service method `Service.Cancel(ctx, req CancelRequest) (Round, error)` in `service.go`:
  - Loads round; rejects if terminal (return `ErrRoundAlreadyTerminal`).
  - If `round.RunID != ""` and `s.canceller != nil`: best-effort `canceller.StopRun(ctx, round.RunID)` (log on error, don't fail — the user already wants it gone).
  - Append a synthetic agent message: `"cancelled by user"` (or include `req.Rationale` if provided).
  - Set `Status = RoundStatusDismissed`, `Decision = {Kind: dismiss, Rationale, DecidedBy, DecidedAt: now}`, `RunID = ""`.
  - `store.SaveRound`.
  - `lock.Release(initiativeName, runID)` — idempotent if already released.
- New handler `Handler.Cancel` in `handler.go` mirroring `Dismiss`'s shape; route `POST /api/v1/initiatives/{name}/feedback/{round}/cancel`. Body: `{rationale?, decided_by?}`.
- Register route in `RegisterRoutes`.
- Tests: cancel happy-path; cancel with nil canceller; cancel terminal round → 409; cancel when lock already released; cancel when `RunID == ""`.

### Phase 3 — API: Stuck-round sweeper
- New file `api/internal/feedback/sweeper.go` with `Sweeper{store, lock, poller, canceller, maxAge time.Duration, interval time.Duration, clock}`.
- Method `Sweeper.RunOnce(ctx)` walks every initiative directory, lists `agent_thinking` rounds, and for each:
  - If round age (`now - UpdatedAt`) > `maxAge`, OR `lock.Inspect` returns nil (lock gone), invoke `Service.Cancel` with rationale `"auto-dismissed: agent run timed out (>30m in agent_thinking)"` or `"auto-dismissed: lock no longer present"`.
- Method `Sweeper.Start(ctx)` runs `RunOnce` on a ticker.
- Wire from `routes_feedback.go` setup: start sweeper goroutine alongside service init; configurable via env `SWARM_MANAGER_FEEDBACK_STUCK_MAX_AGE` (Go `time.Duration`, default `30m`) and `SWARM_MANAGER_FEEDBACK_SWEEP_INTERVAL` (default `5m`).
- Run `RunOnce` once **synchronously at startup** so the production wedged rounds clear on the first deploy.
- Tests: sweeper dismisses old round; sweeper leaves fresh round alone; sweeper releases lock; sweeper safe to run with no initiatives.

### Phase 4 — Existing Dismiss upgrade
- In `service.go:Decide`, when `kind == dismiss` and the round was in `agent_thinking`:
  - Best-effort `canceller.StopRun(round.RunID)`.
  - `lock.Release(initiativeName, round.RunID)`.
- This makes the existing UI **Dismiss** path (still reachable from `Decide`) also stop the agent, even though we expect users to click the new Cancel button. Defense in depth.

### Phase 5 — TS / UI
- `ui/src/types.ts`: add `last_polled_at?: string`, `last_poll_error?: string` to `FeedbackRound`.
- `ui/src/services/feedback-service.ts`: add `cancel(initiative, round, body?: {rationale?, decided_by?}): Promise<FeedbackRound>` calling the new endpoint.
- `feedback-round-card.tsx`:
  - In the `isActive` branch (lines 215–220), render the existing spinner row plus:
    - A small destructive-style **Cancel** button.
    - If `round.last_poll_error` is set: a muted warning line "Agent unreachable: {error}".
  - Wire `useMutation` `cancelMutation` analogous to `decideMutation`. On success, invalidate the same query keys and call `onChanged()`.
  - Add data-testids: `selectors.feedback.cancelButton`, `selectors.feedback.pollErrorNotice` (add to `consts/selectors.ts`).
- Add unit tests with `@testing-library/react` for the card: stuck round shows Cancel button; click triggers service.cancel; success collapses to dismissed view.

### Phase 6 — CLI command (parity)
- In `cli/domains/initiatives/register.go`, add:
  - `support.APICommand("feedback-cancel", "Cancel an in-flight feedback round (--name NAME --round N [--rationale MSG] [--decided-by WHO])", deps.InitiativesFeedbackCancel)`.
- Implement `deps.InitiativesFeedbackCancel` in the matching `domains/initiatives/feedback.go` (or wherever the existing `InitiativesFeedbackDecide` lives), mirroring the decide-dismiss handler. Default human output (per `feedback_cli_default_human_output`).
- Add a CLI integration test that exercises cancel against a fake API server, mirroring the pattern used by `feedback-decide`.

### Phase 7 — Docs & memory
- Update `scenarios/swarm-manager/docs/` (architecture or feedback-specific docs) with the new lifecycle diagram including the cancel + sweep transitions.
- No memory file changes — this is implemented behavior, derivable from code.

## 8. Contract Decisions

### HTTP

`POST /api/v1/initiatives/{name}/feedback/{round}/cancel`

Request body (optional, `application/json`):
```json
{ "rationale": "string?", "decided_by": "string?" }
```

Responses:
- `200 OK` — returns the updated `Round` JSON (now `status: "dismissed"`, `decision.kind: "dismiss"`).
- `404 Not Found` — round doesn't exist (mirror `Dismiss`).
- `409 Conflict` — `{"error": "round is already terminal"}` if status ∈ {applied, rejected, dismissed}.
- `400 Bad Request` — malformed body.

### Round shape additions

```go
type Round struct {
    // ... existing fields
    LastPolledAt     string `json:"last_polled_at,omitempty"`
    LastPollError    string `json:"last_poll_error,omitempty"`
    PollFailureCount int    `json:"poll_failure_count,omitempty"`
}
```

`PollFailureCount` is incremented on each consecutive non-terminal poller error, cleared on success or terminal advance. After 3 consecutive failures, `EnsurePolledTurn` synthesizes a failure-terminal turn.

### Sweeper config

| Env var | Default | Purpose |
|---|---|---|
| `SWARM_MANAGER_FEEDBACK_STUCK_MAX_AGE` | `30m` | Age past which an `agent_thinking` round is force-dismissed |
| `SWARM_MANAGER_FEEDBACK_SWEEP_INTERVAL` | `5m` | How often the sweep ticker runs |
| `SWARM_MANAGER_FEEDBACK_POLL_FAILURE_THRESHOLD` | `3` | Consecutive poll failures before synthesizing a terminal failure |

### Cancel-vs-Dismiss semantics

Both result in `status=dismissed`; both call `StopRun` and `Release` when the round was active.
- `Cancel` is intended for the **stuck `agent_thinking` case** — UI shows a Cancel button only here.
- `Dismiss` (via existing Decide) remains for the **`awaiting_user` case** — the user is choosing not to act on the agent's proposal.

### CLI

```
swarm-manager initiatives feedback-cancel --name <NAME> --round <N> [--rationale TEXT] [--decided-by WHO]
```

Default human output. `--json` for scripting.

## 9. Testing Plan

All tests are automated (per `feedback_testing_over_manual`). No manual test checklists.

### Unit (Go)
- `service_test.go`:
  - `TestService_Cancel_HappyPath` — round in agent_thinking → dismissed, StopRun called, lock released.
  - `TestService_Cancel_NilCanceller` — succeeds; lock released anyway.
  - `TestService_Cancel_TerminalRound` — returns `ErrRoundAlreadyTerminal`.
  - `TestService_Cancel_NoRunID` — succeeds; StopRun not called.
  - `TestService_Cancel_LockAlreadyReleased` — Release returns nil; cancel succeeds.
  - `TestService_Decide_DismissActive_StopsRun` — Phase 4 upgrade.
- `ensure_polled_test.go`:
  - `TestEnsurePolledTurn_PollerErrorRecorded` — sets `LastPollError`, increments counter.
  - `TestEnsurePolledTurn_ThirdConsecutiveFailureSynthesizesFailure` — round transitions to awaiting_user with failure body.
  - `TestEnsurePolledTurn_NotFoundIsTerminal` — `not_found` status mapped to failure.
- `sweeper_test.go` (new):
  - `TestSweeper_DismissesOldRound`
  - `TestSweeper_LeavesFreshRoundAlone`
  - `TestSweeper_DismissesOrphanLockless`
  - `TestSweeper_RunOnceIdempotent`
- `handler_test.go`:
  - `TestHandler_Cancel_200`
  - `TestHandler_Cancel_404`
  - `TestHandler_Cancel_409Terminal`

### Integration (Go)
- Extend `integration_test.go` with a flow: spawn → simulate spawner returning RunID → poller returns `not_found` 3× → sweeper RunOnce → round is dismissed, lock free, new round can start.

### UI (Vitest + React Testing Library)
- `feedback-round-card.test.tsx`:
  - Renders Cancel button when `status === "agent_thinking"`.
  - Click triggers `feedbackService.cancel`; on success the card re-renders with dismissed decision.
  - Renders `last_poll_error` text when present.
  - Cancel button absent for terminal/awaiting_user statuses.

### CLI (Go)
- `feedback_cancel_test.go` mirroring `feedback_decide_test.go`: round-trip against a fake API server.

### Run commands

```bash
cd scenarios/swarm-manager && make test                 # full scenario suite
cd scenarios/swarm-manager/api && go test ./... -timeout 300s
cd scenarios/swarm-manager/cli && go test ./...
cd scenarios/swarm-manager/ui && npm run test -- feedback
```

## 10. Rollout / Validation Checklist

Per `feedback_planning_guidelines`, the plan must (a) fix all current issues including pre-existing wedged rounds, and (b) restart the scenario. Per `feedback_use_vrooli_scenario_restart`, restart via `vrooli scenario restart swarm-manager`.

Checklist (must be green before declaring done):

1. `make test` in `scenarios/swarm-manager` passes.
2. `vrooli scenario restart swarm-manager` succeeds.
3. After restart: the existing stuck round on initiative `command-center-foundation` (round 1) is in `dismissed` status with rationale containing `auto-dismissed`. Verify via:
   ```bash
   swarm-manager initiatives feedback-list --name command-center-foundation
   swarm-manager initiatives feedback-get  --name command-center-foundation --round 1
   ```
4. `vrooli scenario logs swarm-manager --tail` shows the startup sweep log line with at least 1 round dismissed.
5. New cancel flow exercised end-to-end automatically by integration test.
6. Lock file for that initiative is gone (not just rewritten):
   ```bash
   ls scenarios/swarm-manager/data/initiatives/command-center-foundation/.lock 2>&1
   ```
7. Submitting a fresh feedback round on `command-center-foundation` succeeds (proving lock is released and round folder doesn't have a lingering active round).

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Sweeper force-dismisses a healthy long-running agent run that legitimately took >30m | Low | Medium | Default 30m is generous (current agent-manager runs cap well below this); ship env override; sweeper checks the poller first and only synthesizes failure if the run is genuinely unreachable, not just slow |
| Cancel races with `RecordAgentTurn` (agent reports back exactly as user clicks Cancel) | Low | Low | `Cancel` rejects terminal rounds with 409; both paths converge on `RoundStatusDismissed`/`RoundStatusAwaitingUser` and persist via the same `store.SaveRound`. Last-write wins; user reload reflects truth. |
| `StopRun` against an already-dead agent-manager run errors | High | Low | Best-effort: log and continue; cancel still succeeds locally |
| Sweeper goroutine panics and dies silently | Low | Medium | Wrap `RunOnce` in `recover()`; log and continue ticker |
| `PollFailureCount` triggers spuriously on transient agent-manager hiccups | Medium | Low | Threshold of 3 consecutive failures + 30m wall-clock backstop means a hiccup followed by recovery doesn't false-positive |
| New fields break older round JSON files on disk | Low | Low | All new fields are `omitempty` and pointer/zero-value tolerant; `store.LoadRound` already tolerates unknown-field-free JSON |

## 12. Non-goals / Prohibited Patterns

- **Do not** add a "retry the agent" button in this plan — out of scope; track separately. Cancel + new feedback round is the supported retry shape.
- **Do not** introduce a webhook callback from agent-manager to swarm-manager. Polling stays canonical; the sweeper is the safety net.
- **Do not** add backwards-compatibility shims for the old `Round` shape. Greenfield: round JSONs are forward-compatible because all new fields are `omitempty`.
- **Do not** bypass the CLI when adding scripts/skills around this feature. Per `feedback_skills_use_cli_never_api`, every consumer outside the API itself uses the new `feedback-cancel` CLI command, never raw HTTP.
- **Do not** skip the CLI command. Per `cli-steer`, every API endpoint has a CLI counterpart.
- **Do not** modify the `Decide` shape; only its internal behavior for the dismiss-while-active case (Phase 4).
- **Do not** persist `LastPollError` for transient successes — clear it on terminal advance and on poller success-but-non-terminal.

## 13. Definition of Done

A future agent can resume from this plan and declare done when **all** of the following are true:

- [ ] `POST /api/v1/initiatives/{name}/feedback/{round}/cancel` exists, returns 200 on a stuck round, 404 on missing, 409 on terminal.
- [ ] `Round` JSON includes `last_polled_at`, `last_poll_error`, `poll_failure_count` (omitempty).
- [ ] `EnsurePolledTurn` synthesizes a failure terminal turn after `SWARM_MANAGER_FEEDBACK_POLL_FAILURE_THRESHOLD` consecutive poller errors.
- [ ] Background sweeper running on a ticker; force-dismisses rounds older than `SWARM_MANAGER_FEEDBACK_STUCK_MAX_AGE` or whose lock is gone.
- [ ] Synchronous `RunOnce` invoked at API startup; verified to clear existing wedged rounds on first deploy.
- [ ] `Decide(kind=dismiss)` against an `agent_thinking` round calls `StopRun` and releases the lock.
- [ ] UI: `feedback-round-card.tsx` `isActive` branch renders a working Cancel button; `last_poll_error` surfaced when present.
- [ ] TS service `feedbackService.cancel(...)` exists and is typed.
- [ ] CLI: `swarm-manager initiatives feedback-cancel --name N --round R` works; default human output; `--json` supported.
- [ ] All listed unit, integration, and UI tests added and passing.
- [ ] `vrooli scenario restart swarm-manager` succeeds.
- [ ] On the live instance, `command-center-foundation` round 1 is `dismissed` post-deploy, lock file gone, a new feedback round can be submitted.
- [ ] Plan §10 checklist items 1–7 all green.
