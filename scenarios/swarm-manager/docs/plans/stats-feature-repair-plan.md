# Swarm-Manager Stats Feature Repair — Implementation Plan

## 1. Purpose

The swarm-manager UI's Stats panel (Dashboard / Throughput / Agent / Timing tabs) presents misleading numbers: 100% agent failure rate despite 26 completed backlog items, lead time where average equals median exactly because there is only one valid sample, cycle time `<1 min` that is actually an empty-sample artifact, and zero workshop rounds because the emitter was never wired.

This plan repairs the underlying event-collection gaps, reshapes metrics that were instrumented against the wrong lifecycle, and adds UI affordances so legitimately sparse data does not look like a product malfunction.

## 2. Required Reading

Run at the start of any execution session:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement implementation-plan-authoring react-coherence test
```

## 3. Problem Statement

Direct queries against the event log at `~/.local/share/vrooli/swarm-manager/events.db` (2,694 events total) confirm every suspicious number observed in the UI:

| Symptom | Root-cause evidence | Classification |
|---|---|---|
| Agent tab: 0% success, 100% failure, 39 executions | 0 `execution.completed` events in DB despite 26 completed backlog items. 35 executions terminate in `validating` state because `finishFinalization` in `internal/execution/finalization.go:238-243` and all 6 sites in `finalization_store.go` call `dispatchStatusUpdate` but never `logExecutionEvent`. | Collection bug |
| Agent tab: 22/26 completed backlog items came via `failed → completed` | User-initiated override path: agent self-reports failed, user reviews and accepts the work. Stats currently count the associated execution as permanently failed. | Modeling gap |
| Timing tab: lead time avg == median (1.1 hrs) | Only **1** entity has both a `backlog.created` event and a later `backlog.status_changed→completed` event. `EmitBacklogCreated` in `handler_create.go:218` was wired after most pre-existing items were already in the system. | Historical data loss |
| Timing tab: cycle time `<1 min` | `engine.go:196-199` records cycle time from `in_progress → completed` on backlog items, but backlog items **never** transition to `in_progress` (0 events in DB). The slice is empty; `avgFloat([])` returns `0` which the UI formatter renders as `<1 min`. | Wrong metric concept |
| Timing tab: queue wait `<1 min` | Same pattern: `engine.go:186-189` requires a `queued` transition, and backlog items rarely use that state. | Wrong metric concept |
| Agent tab: Avg Workshop Rounds 0.0 | `EmitWorkshopRoundCompleted` is defined on the emitter interface (`internal/backlog/handler.go:79`) but **never called** anywhere in production code. Zero `decision.workshop_round_completed` events exist. | Emitter not wired |
| Dashboard: velocity sparse, 30d completions == all-time | Legitimate — the app has less than 30 days of history — but the UI does not distinguish this from a real problem. | UI affordance gap |
| All tabs: no denominator / sample size visible | Rates computed from zero-count denominators render identically to rates computed from large samples. | UI affordance gap |

## 4. Scope

**In scope**
- Emit `execution.completed` on the `validating → completed` transition and all other missed finalization status changes.
- Introduce a distinct concept for "manually accepted" executions (user overrides a failed run) so those count toward throughput and success, not failure.
- Wire `EmitWorkshopRoundCompleted` into the actual workshop-round-completion call site.
- Reshape cycle-time and queue-wait metrics to match the lifecycle that actually exists, or remove them if no meaningful signal exists.
- Backfill a one-time migration that synthesizes missing `execution.completed` events for executions whose finalization already ran successfully before the fix.
- Add a `StatsMetricCard` / `InsufficientData` composite to the UI that shows sample-size denominators and an explicit "not enough data yet" state.
- Surface a global "history window" banner on the Stats panel when total event history is shorter than the largest selected window (e.g., 30d).
- Fix all lint / type / unit-test issues in modified files, including pre-existing ones.

**Out of scope**
- Replacing the in-memory watermark stats engine with a database-backed aggregate store.
- Adding new metric tabs beyond what already exists.
- Retroactively inventing `backlog.created` events for items that existed before the emitter was wired (the data is gone; the UI simply honors this with a denominator).
- Changing the underlying execution state machine.

**Greenfield constraint:** No compatibility shims, no dual-write of old and new event shapes, no legacy fallbacks. The one-time backfill is a migration, not a compatibility layer — it runs once at startup when a sentinel key is absent, then never again.

## 5. Current Technical Context

**API**
- Event log (SQLite): `~/.local/share/vrooli/swarm-manager/events.db`, schema in `internal/eventlog/repository.go`, types in `internal/eventlog/types.go`.
- Event emitter: `internal/eventlog/emitter.go` (implements interfaces declared at each call site, e.g. `internal/execution/interfaces.go:58-62`, `internal/backlog/handler.go:69-79`).
- Stats engine: `internal/stats/engine.go` (watermark pattern — `Rebuild` on startup, `Refresh` per request), `internal/stats/metrics.go` (aggregate → response shaping), `internal/stats/types.go` (response types), `internal/stats/handler.go` (HTTP handler).
- Execution lifecycle: `internal/execution/service.go:315-338` (`logExecutionEvent`), `internal/execution/polling.go:160-240` (polling-driven transitions), `internal/execution/finalization.go:193-260` (`finishFinalization`), `internal/execution/finalization_store.go` (store-driven transitions).
- Backlog lifecycle: `internal/backlog/handler.go:360-400` (`EmitBacklogStatusChanged` at line 383), `internal/backlog/handler_create.go:218` (`EmitBacklogCreated`).
- Workshop: `internal/backlog/workshop.go` and round handlers (location of round-completion not yet wired to emitter).

**UI**
- Stats panel: `ui/src/surfaces/graph/components/StatsPanel.tsx` (6 tabs).
- Hook: `ui/src/surfaces/graph/hooks/useStats.ts` (60s refetch).
- Format helpers: `ui/src/lib/stats-format-utils.ts`.
- Existing empty-state pattern: `StatsPanel.tsx:179-180, 327-340, 357-358` (Blocking/Scope tabs — pattern to reuse).

**Confirmed data (events.db snapshot at plan authoring time)**
- 2,694 events total, earliest ~2026-04-10, so the "30d window == all-time" observation is legitimate history length, not pruning.
- 39 `execution.created`, 11 `execution.failed`, 3 `execution.canceled`, 0 `execution.completed`, 96 `execution.status_changed` (last-status breakdown: 35 validating, 11 failed, 2 canceled, 1 running).
- 35 `backlog.created`, 30 `backlog.status_changed` (22 failed→completed, 2 completed→archived, 2 backlog→completed, 2 backlog→archived, 1 queued→completed, 1 archived→completed). Zero transitions through `in_progress`.
- 0 `decision.workshop_round_completed` events.

## 6. Target End State

- Every execution that reaches a terminal status (`completed`, `failed`, `canceled`, `needs_fixup`) emits exactly one matching event into the event log, from every path that can reach that status.
- "Manual accept" — a backlog `failed → completed` transition — resolves the associated execution(s) to `StatusCompleted` with a recorded `ManuallyAccepted` flag, and emits `execution.manually_accepted` in addition to `execution.completed`. Both throughput and success-rate reflect this.
- Workshop rounds emit `decision.workshop_round_completed` when the round file is persisted.
- Cycle time and queue wait either reflect real lifecycle timestamps (backlog `created → completed` split by agent work intervals if we can reconstruct them, or execution `running → completed`) or are removed from the Timing tab and replaced with a metric that has actual samples. The Timing tab never shows `<1 min` for an empty sample set — empty sets render as "No data yet".
- Every metric card in the Stats panel shows an explicit sample-size denominator (e.g., `(3 of 39 executions reviewed)`). When a metric is zero-sample or below a minimum-meaningful threshold, it renders the shared `InsufficientDataCard` with a reason string instead of a numeric zero.
- The Stats panel shows a one-line history-window banner at the top when the oldest event is within the currently selected lookback window.
- Restart of the swarm-manager scenario leaves the stats engine with fully-populated aggregates after a one-time backfill migration.

## 7. Implementation Strategy (phased)

### Phase 1 — Close the execution event-emission gap (API)

**Goal:** Zero terminal-status execution transitions happen without a matching log event.

1. Audit every production call site that mutates `record.Status` to a terminal value. Inventory sites identified so far:
   - `internal/execution/polling.go:167` (already emits at line 233 — keep as baseline).
   - `internal/execution/finalization.go:220, 224, 229, 234` (no emitter).
   - `internal/execution/finalization_store.go:35, 73, 130, 158, 178, 220` (no emitter).
   - `internal/execution/service_control.go:151, 177, 197` (already emits — keep).
2. At each site that mutates status, capture `prevStatus := record.Status` before the mutation, then after a successful `store.Save`, call `s.logExecutionEvent(*record, prevStatus)` exactly once.
3. Extract a small helper `s.applyTerminalStatus(record *Record, next Status, prev Status)` that owns (a) status assignment, (b) `FinishedAt` stamp, (c) save, (d) `logExecutionEvent`, (e) `dispatchStatusUpdate`. Replace the six duplicated blocks with calls to this helper. This removes the class of bug by construction; missing the emitter again would require someone to bypass the helper.
4. Add a Go-level assertion in `logExecutionEvent` that rejects a call where `prevStatus == record.Status`, panicking in tests (via a build-tag guard) and logging+noop in production. Future drift shows up loudly.

### Phase 2 — Manual-accept path as a first-class concept

**Goal:** "Agent said failed, user said good enough" produces a successful outcome in the stats, not a permanent failure.

1. Add `ManuallyAccepted bool` and `AcceptedBy string` fields to `execution.Record`, and an `execution.manually_accepted` event type in `internal/eventlog/types.go` with payload `{AcceptedBy, Reason}`.
2. In the backlog status-change handler (`internal/backlog/handler.go:360-400`), when a transition is `failed → completed`:
   - Find the latest non-terminal-cancel execution(s) for that backlog item via the execution service.
   - For each, call a new `execution.Service.ManuallyAccept(execID, acceptor, reason)` which mutates status to `StatusCompleted`, sets `ManuallyAccepted = true`, stamps `FinishedAt`, emits `execution.manually_accepted`, then calls the same `applyTerminalStatus` path from Phase 1 (which emits `execution.completed`).
3. In `internal/stats/engine.go`, process `execution.manually_accepted`:
   - Increment a new aggregate `execManuallyAccepted` counter.
4. In `internal/stats/metrics.go:buildAgent`, redefine:
   - `finished = execCompleted + execFailed` (unchanged — `execCompleted` now also includes manually-accepted).
   - Add a new response field `AgentStats.ManualAcceptRate = execManuallyAccepted / finished`.
   - Success rate stays honest (`execCompleted` includes manual accepts), but the UI can break it down.
5. UI shows "Success rate X% (of which Y% manually accepted after agent flagged failure)" so manual judgment is visible.

### Phase 3 — Wire workshop round emission

**Goal:** `decision.workshop_round_completed` is emitted whenever a round file is persisted.

1. Find the workshop round write site (likely in `internal/backlog/workshop.go` or a neighboring file). Grep `round-%03d.json` as an anchor.
2. In that write path, after the round file is persisted, call `h.eventLogger.EmitWorkshopRoundCompleted(entityID, roundNumber)`. Thread the emitter dependency in if it's not already available.
3. If rounds can be deleted (`WorkshopDeleteRound` handler at `handler.go:240`), emit a corresponding `decision.workshop_round_deleted` event so the `workshopRounds` aggregate can decrement — or, simpler, have the stats engine recompute max round per entity from live state rather than from event delta. Choose the simpler path.

### Phase 4 — Reshape wrong-lifecycle timing metrics

**Goal:** Timing tab shows metrics that actually have samples.

1. Decide by inspection, then document in Contract Decisions (§8):
   - Drop the backlog `in_progress`-based cycle time metric. Backlog items don't have that state.
   - Replace the Timing tab's Cycle Time card with **Execution Duration**, computed from `execution.created → execution.completed` of each execution record — which we already record via `p.DurationSeconds` in `ExecutionCompletedPayload`. This reuses the existing `execDurations` slice in the stats engine.
   - Keep Lead Time as-is (created → completed on backlog items) but expose its sample size.
   - Drop Queue Wait. Backlog items don't use a `queued` state in practice, and the value is always `<1 min` for the same empty-slice reason. When a real queue system appears, this can come back.
2. Update `StatsResponse.Timing` shape accordingly — delete `AvgCycleTimeHours`/`MedianCycleTimeHours`/`AvgQueueWaitHours`, add `ExecutionDuration{Avg,Median}Minutes` reading from `execDurations`.
3. Update the UI Timing tab to match.

### Phase 5 — Surface denominators and sample sizes in the API

**Goal:** Every rate and every duration metric ships with the count of observations behind it.

1. Augment each stats response substructure with `SampleSize int` fields. Examples:
   - `AgentStats.SuccessRateSampleSize = execCompleted + execFailed`
   - `TimingStats.LeadTimeSampleSize = len(leadTimesH)`
   - `TimingStats.ExecutionDurationSampleSize = len(execDurations)`
   - `DashboardStats.VelocityWeeksCovered = howManyNonZeroWeeks`
   - `DashboardStats.HistoryDays = (now - earliestEventTimestamp).Days()`
2. Expose the earliest-event timestamp in the top-level response so the UI can render the history banner without a second query.

### Phase 6 — One-time backfill for stuck validating executions

**Goal:** The 35 executions currently stuck at `validating` with a completed finalization get their missing `execution.completed` event synthesized once.

1. Add a startup migration in `internal/stats` (or a sibling `internal/migrations`) keyed by a sentinel event `system.migration_applied` with name `backfill_execution_completed_v1`.
2. Migration logic: iterate all executions in the store. For each where the record's `Status == Completed` but no `execution.completed` event exists in the log, emit a synthetic `execution.completed` with its `FinishedAt` as the event timestamp and `DurationSeconds` computed from `StartedAt → FinishedAt`. For each where `Status == Validating` and `Finalization.Status == Completed` — treat as completed, set the record's status to `Completed`, and emit both the terminal-status-changed event and `execution.completed`.
3. Record the sentinel so subsequent restarts skip this.
4. After emission, the watermark engine's next `Rebuild` picks up the new events automatically.

### Phase 7 — UI: shared `InsufficientDataCard` + denominators

**Goal:** No tab renders a misleading zero.

1. Create `ui/src/shared/ui/composites/InsufficientDataCard.tsx`:
   - Props: `{ title: string, reason: string, required?: number, have?: number }`.
   - Renders a styled card with a neutral icon, title, and reason like "Need ≥5 completed executions to estimate this. Currently 1."
2. Create `ui/src/shared/ui/composites/StatsMetricCard.tsx`:
   - Props: `{ label, value, sampleSize, unit?, formatter? }`.
   - Shows the value prominently, with `(n of N)` in muted text under it.
   - If `sampleSize < 1`, delegates render to `InsufficientDataCard` with a default reason.
3. Refactor the Dashboard, Throughput, Agent, Timing tabs in `StatsPanel.tsx` to use `StatsMetricCard` for every card. Map each card to the API's new `SampleSize` field.
4. Add a `HistoryBanner` composite and render it at the top of `StatsPanel.tsx` when `response.historyDays < max(selectedWindowDays)`. Copy: "Stats window is 30 days — data only goes back {historyDays} day(s). Metrics will stabilize with more history."
5. For the Dashboard Velocity chart, when `velocityWeeksCovered < 4`, render the chart with a subdued "(not enough history to estimate remaining time)" caption instead of the Est. Remaining pill.

### Phase 8 — Cleanup & verification

1. Run `gofumpt -w ./internal/...` and `golangci-lint run ./...` in `scenarios/swarm-manager/api`. Fix every warning in modified files, **including pre-existing ones**.
2. Run `go build ./...` and `go test ./... -timeout 300s` in the API. Fix every failure.
3. Run `npx tsc --noEmit` and `eslint` in `scenarios/swarm-manager/ui`. Fix every warning in modified files, **including pre-existing ones**.
4. Run `npm run test` in the UI. Fix every failure.
5. **User (not agent) runs `vrooli scenario restart swarm-manager`** and verifies via `curl -s http://localhost:<port>/health` and by opening the Stats panel.

## 8. Contract Decisions

- **Terminal transition helper**: `applyTerminalStatus(record, next, prev)` becomes the only supported way to set a terminal execution status. All existing manual assignments are rewritten to route through it.
- **`execution.manually_accepted` event**: A manually-accepted execution emits *both* `execution.manually_accepted` and `execution.completed`. The completed event carries `HadFixups=false` and a `ManuallyAccepted=true` flag in its payload so stats downstream can distinguish agent-finished from human-finished.
- **Cycle Time → Execution Duration**: The Timing tab's "Cycle Time" card is renamed and rebased. Backend response field renamed from `AvgCycleTimeHours`/`MedianCycleTimeHours` to `AvgExecutionDurationMinutes`/`MedianExecutionDurationMinutes`. The UI label changes too.
- **Queue Wait removed**: Field removed from response; UI card removed.
- **Sample size fields**: All rate-based and duration-based metrics gain a sibling `*SampleSize int` field in the API response.
- **Backfill sentinel**: A dedicated event type `system.migration_applied` with `{name: "backfill_execution_completed_v1"}` gates re-runs. Idempotent.
- **Empty-state UX contract**: A metric with `sampleSize == 0` renders an `InsufficientDataCard`. A metric with `0 < sampleSize < 5` renders the value with a low-confidence visual hint (muted color + sample size). `5` is a placeholder threshold; treat it as a tunable constant exposed from the engine response.

## 9. Testing Plan

**All verification is automated. No manual test checklists.**

API unit tests (`scenarios/swarm-manager/api/internal/stats/*_test.go` and `internal/execution/*_test.go`):

1. **`TestExecutionCompletedEventEmittedFromFinalization`** — set up an execution, drive it through `finishFinalization` with a successful finalization, assert the event log contains exactly one `execution.completed` event.
2. **`TestExecutionCompletedEventEmittedFromFinalizationStore`** — parameterized test covering all 6 sites in `finalization_store.go`.
3. **`TestManualAcceptEmitsBothEvents`** — user-accept path: assert both `execution.manually_accepted` and `execution.completed` appear, and that stats success-rate reflects the accept.
4. **`TestApplyTerminalStatusRejectsNoOp`** — helper rejects a transition where `prev == next`.
5. **`TestWorkshopRoundCompletedEventEmitted`** — after a round file is written, the event appears in the log.
6. **`TestBackfillEmitsMissingCompletedEvents`** — seed a store with the exact pattern from the current DB (35 validating-with-finalization-completed) and assert the backfill migration emits one `execution.completed` per stuck record, then skips on second run because the sentinel is present.
7. **`TestStatsEngineExposesSampleSizes`** — call the stats API with known event distribution, assert `SampleSize` matches the slice lengths behind each metric.
8. **`TestLeadTimeSampleSizeSingleton`** — seed exactly one paired created→completed and assert `LeadTimeSampleSize == 1` (avg == median is then a correct observation, not a bug).
9. **`TestExecutionDurationReplacesCycleTime`** — assert Timing response has `AvgExecutionDurationMinutes`/`Median...` and no longer has `AvgCycleTimeHours`.

UI unit tests (`scenarios/swarm-manager/ui/src/**/__tests__/*.test.tsx`):

10. **`StatsMetricCard.renders-value-with-sample-size`**, **`.delegates-to-insufficient-when-zero`** — snapshot + assertion tests.
11. **`StatsPanel.history-banner-appears-when-history-short`** — render with mocked API response where `historyDays=5` and `selectedWindowDays=30`.
12. **`StatsPanel.agent-tab-shows-manual-accept-breakdown`** — render with mocked API response containing non-zero `manualAcceptCount`.

API integration test (testcontainers or in-process):

13. **`TestStatsEndToEndAfterRealExecution`** — drive a fake backlog item through create → execute → validating → finalized-completed, then hit `/api/v1/stats` and assert `AgentStats.SuccessRate == 1.0`, `TotalExecutions == 1`, `CompletedAllTime == 1`, `LeadTimeSampleSize >= 1`.

**Regression guard**: Test #1 and #2 would have failed on the current `master` and are the durable regression tests for the observed production bug.

## 10. Rollout / Validation Checklist

After implementation, the user restarts swarm-manager. Then they verify:

- `curl -s http://localhost:<swarm-manager-port>/api/v1/stats | jq '.agent'` returns a success rate that reflects the 22 previously-failed-then-manually-completed items plus any net-new successful runs — **not** the 0% shown today.
- `... | jq '.timing.leadTimeSampleSize'` returns > 0 and — post-any-new-completions — > 1.
- `... | jq '.timing.avgExecutionDurationMinutes'` is populated (derived from existing execution durations in the aggregate, which already has 11 samples from failed runs).
- The Stats panel opens without errors and every card either shows a value with `(n of N)` or an `InsufficientDataCard`.
- The history banner appears at top of the Stats panel and disappears after 30+ days of history exist.

## 11. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Adding `logExecutionEvent` to finalization paths double-emits if polling and finalization both fire. | Idempotent guard: `logExecutionEvent` already short-circuits when `prevStatus == record.Status`. Route all terminal transitions through `applyTerminalStatus` so the prevStatus check is always in place. Test #4 enforces this. |
| Manual-accept event model collides with the existing fixup workflow (which also ends a failed run with a new successful run). | Fixup is a new execution record; manual-accept mutates the existing one. These are orthogonal. Document in Contract Decisions and cover both in tests. |
| Backfill migration runs on a fresh DB and emits nothing — but runs unnecessarily on every restart. | Sentinel event gates re-run. Test #6 covers both paths. |
| Removing Queue Wait and Cycle Time fields breaks an external consumer. | No external consumers exist; only the UI in this repo reads the stats endpoint. Grep confirms. |
| UI denominator rendering clutters the cards on mobile. | `StatsMetricCard` uses a muted secondary line; test #10 runs under a mobile viewport to confirm. |
| Workshop round emitter is called at the wrong point in the round lifecycle (e.g., before the file is fully written). | Emit after `os.WriteFile` returns success. Test #5 validates both the success and the write-error branches (no emit on write failure). |

## 12. Non-goals / Prohibited Patterns

- **No compatibility shims.** Remove `AvgCycleTimeHours` and `AvgQueueWaitHours` outright — do not keep them as deprecated mirrors.
- **No manual test checklists.** Every verification step is an automated test in §9.
- **No silent fallbacks.** A metric with no samples must render `InsufficientDataCard`, not `0`.
- **No data invention.** Do not fabricate `backlog.created` events for pre-existing items. The backfill in Phase 6 is scoped strictly to executions whose terminal status is already known from the store.
- **No scenario restart by the agent.** The agent writes code to disk and runs tests. The user runs `vrooli scenario restart swarm-manager`.

## 13. Definition of Done

- [ ] Every production site that mutates execution status to a terminal value goes through `applyTerminalStatus`.
- [ ] `execution.completed` is emitted from `validating → completed`, `needs_fixup → completed`, and every other terminal path (verified by tests #1, #2).
- [ ] `execution.manually_accepted` emitted on backlog `failed → completed`; associated execution record updated (#3).
- [ ] `decision.workshop_round_completed` emitted on round persistence (#5).
- [ ] Backfill migration written, gated, tested (#6).
- [ ] Stats API response includes `SampleSize` on every rate/duration metric and `historyDays` at the top level (#7, #8).
- [ ] Timing tab uses execution duration, not cycle time; queue wait removed (#9).
- [ ] `InsufficientDataCard` and `StatsMetricCard` composites exist in `shared/ui/composites/` and are used by every Stats panel tab (#10).
- [ ] History banner renders when history is shorter than selected window (#11).
- [ ] `go build ./...`, `go test ./...`, `golangci-lint run`, `gofumpt -l` all clean in the API.
- [ ] `npx tsc --noEmit`, `eslint`, `npm run test` all clean in the UI.
- [ ] All warnings and failures in touched files are resolved, including pre-existing ones.
- [ ] User has restarted swarm-manager and confirmed `/api/v1/stats` returns the expected shape and values.
