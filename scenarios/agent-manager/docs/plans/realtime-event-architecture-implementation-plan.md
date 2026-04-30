# Agent Manager Realtime Event Architecture — Implementation Plan

## 1. Purpose

Make agent-manager's run timeline and control surface production-ready by turning the current "REST plus best-effort WebSocket refetch" behavior into a single, testable realtime event pipeline.

This plan implements the prior investigation recommendations:

- One client-side run event store instead of split global/list/detail realtime logic
- Resumable WebSocket delivery using persisted event sequence and REST gap-fill
- Shared lifecycle handling for normal runs, stop, and continue
- Authoritative event append-before-broadcast semantics
- Action flags and endpoint responses that stay consistent after mutations
- Focused unit/integration tests at the event, lifecycle, WebSocket, and UI seams

No code changes have been made for this plan.

## 2. Required Reading

Run these before implementation:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health react-coherence interoperability-steer unit-testing-architecture-steer seam-discovery-and-enforcement utils-unification
prompt-manager skill read cli-steer api-steer visited-tracker-tools knowledge-observatory-tools
```

Relevant local docs:

```bash
sed -n '1,260p' scenarios/agent-manager/docs/internal/SEAMS.md
sed -n '1,260p' scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md
sed -n '1,260p' scenarios/agent-manager/docs/internal/COHERENCE_NOTES.md
sed -n '1,260p' scenarios/agent-manager/docs/internal/UTILS_UNIFICATION_NOTES.md
```

Evidence commands already run during planning:

```bash
pnpm --dir scenarios/agent-manager/ui run type-check
pnpm --dir scenarios/agent-manager/ui run test:unit
cd scenarios/agent-manager/api && go test ./internal/handlers ./internal/adapters/event ./internal/orchestration -run 'WebSocket|Event|Continue|Stop|RunActions'
```

Observed result: all three checks passed.

## 3. Problem Statement

The backend has a strong event-oriented model, but the UI and orchestration paths currently have several divergent sources of truth:

1. `ui/src/App.tsx` subscribes to all WebSocket events and refetches the run list globally.
2. `ui/src/pages/RunsPage.tsx` also subscribes to selected-run events and keeps its own local `events`, `selectedRun`, `runOverrides`, and reconnect reconciliation logic.
3. `ui/src/hooks/useWebSocket.ts` drops `subscribe` / `unsubscribe` calls when the socket is not open.
4. `api/internal/orchestration/service.go` has a separate continuation lifecycle that duplicates parts of `RunExecutor`.
5. `api/internal/orchestration/terminator.go` updates a stopped run to cancelled but does not provide the same event/status broadcast contract as normal terminal paths.
6. `api/internal/adapters/event/sqlite.go` assigns event sequence with `SELECT MAX(sequence)` and can race under concurrent appenders.
7. `broadcastingEventSink` broadcasts even when append fails, so the UI can observe events that are not durable and later disappear after REST reconciliation.

These gaps create the whack-a-mole pattern: individual fixes patch one stale state path while another path still has different timing, dedupe, or lifecycle assumptions.

## 4. Scope

In scope:

- Backend event append and broadcast reliability
- Stop/continue lifecycle event consistency
- WebSocket client durable subscription intent and reconnect behavior
- UI selected-run timeline state, run-list state, and action state coherence
- Tests for event ordering, reconnect/gap-fill, stop, continue, and timeline rendering inputs
- Documentation updates for seams, temporal flows, and UI coherence notes

Out of scope:

- Replacing WebSocket with SSE or a new transport
- Rebuilding the entire UI state stack with a heavyweight store dependency
- Changing runner codec event semantics except where needed to preserve durable append order
- Changing the public meaning of existing event types
- Adding new dependencies without explicit approval

## 5. Current Technical Context

Key files:

| Area | File | Current role |
|---|---|---|
| Global UI data + WS | `ui/src/App.tsx` | Owns health/profiles/tasks/runs hooks, connects WS, subscribes all, refetches runs on WS messages |
| WS client | `ui/src/hooks/useWebSocket.ts` | Parses proto WS messages, sends subscribe commands, reconnects with backoff |
| Run details page | `ui/src/pages/RunsPage.tsx` | Owns selected run, events, diff, event merge, reconnect refetch, route sync, run actions |
| Timeline transform | `ui/src/lib/runTimeline.ts` | Pure event sorting, categorization, filtering, and tool grouping |
| Timeline UI | `ui/src/components/RunTimeline.tsx` | Renders messages/tools/events and sends continue/delete-message commands |
| WS server | `api/internal/handlers/websocket.go` | Hub, subscription filtering, connected/pong messages, broadcast fanout |
| Event store | `api/internal/adapters/event/sqlite.go` | Durable event append/get/stream with in-memory subscribers |
| Run coordinator | `api/internal/orchestration/run_executor.go` | Normal run phase coordinator and finalization seam |
| Orchestrator service | `api/internal/orchestration/service.go` | Create/stop/continue/delete-message APIs, event sinks, dispatcher sink |
| Terminator | `api/internal/orchestration/terminator.go` | Robust runner/process termination |
| Actions | `api/internal/domain/run_actions.go` | Canonical action flags consumed by UI |

Existing strengths to preserve:

- `RunActionsFor` is already a good decision boundary.
- `RunTimeline` has pure transformation utilities and unit tests.
- `RunExecutor` has an explicit phase/finalize contract.
- Event, runner, sandbox, and phase seams are documented in `docs/internal/SEAMS.md`.

## 6. Target End State

### Backend

- Every event that reaches WebSocket subscribers has first been durably appended or is explicitly marked non-durable.
- Event sequence allocation is safe under concurrent appenders.
- Stop emits the same status event and `run_status` broadcast shape that terminal normal runs emit.
- Continue uses a shared lifecycle/update helper so status changes, heartbeats, timeout behavior, event emission, and action hydration are consistent.
- Mutating run endpoints return hydrated runs/actions where possible, or have a documented reason when a legacy response shape remains.

### Frontend

- `useWebSocket` tracks desired subscriptions and replays them after reconnect.
- A single run event store owns:
  - run snapshots by ID
  - events by run ID
  - last known sequence by run ID
  - selected-run gap-fill via `/runs/{id}/events?after_sequence=N`
  - dedupe by event ID and sequence
  - terminal reconciliation without fixed sleep guesses
- `App` no longer performs broad run-list refetches for every run event.
- `RunsPage` consumes the shared store instead of maintaining ad hoc event merge/reconnect logic.
- `RunTimeline` remains a pure display surface over normalized events.

## 7. Implementation Strategy

### Phase 0 — Baseline and Guardrails

1. Run and save baseline outputs:
   ```bash
   pnpm --dir scenarios/agent-manager/ui run type-check
   pnpm --dir scenarios/agent-manager/ui run test:unit
   cd scenarios/agent-manager/api && go test ./internal/handlers ./internal/adapters/event ./internal/orchestration
   ```
2. Record visited files with `visited-tracker` under tag `agent-manager-realtime-plan`.
3. Add implementation work behind existing behavior first; avoid deleting old UI paths until the new store has unit coverage.

### Phase 1 — Make Event Append Authoritative

Goal: no WebSocket event should disappear after REST reconciliation.

1. Introduce an `event.Appender` helper or method inside `adapters/event` that allocates sequence safely.
2. Fix SQLite sequence allocation. Preferred options, in order:
   - Use a per-run transaction with `BEGIN IMMEDIATE` or equivalent write lock.
   - Add retry-on-unique-conflict around `SELECT MAX(sequence)` + insert.
   - Add a `run_event_cursors` table only if SQLite locking/retry is insufficient.
3. Update `broadcastingEventSink.Emit` so append failure prevents broadcast unless the event is explicitly classified as non-durable.
4. Add a single backend helper for "append and broadcast":
   - Input: `runID`, event(s), broadcaster
   - Output: appended events with assigned IDs/sequences
   - Behavior: append first, then broadcast assigned persisted events
5. Replace direct `o.events.Append(...); o.broadcaster.BroadcastEvent(...)` pairs in `service.go` with the helper.

Tests:

- `adapters/event/sqlite_test.go`: concurrent appenders for one run produce contiguous unique sequences.
- `orchestration/service_test.go`: append failure does not broadcast a run event.
- Existing event retrieval tests still pass.

### Phase 2 — Normalize Run Status Mutation and Broadcast

Goal: every run status mutation follows one decision boundary.

1. Extract a backend helper, likely in `orchestration/run_lifecycle.go`:
   ```go
   type RunStatusTransitionInput struct {
       Run *domain.Run
       NewStatus domain.RunStatus
       Phase domain.RunPhase
       Reason string
       EndedAt *time.Time
       ErrorMsg string
       ExitCode *int
   }
   ```
2. The helper should:
   - update run fields
   - persist run
   - append a `status` event when status changes
   - broadcast appended event
   - broadcast `run_status`
   - return `attachRunActions(ctx, run)`
3. Use this helper in:
   - `StopRun` fallback path
   - `Terminator.StopRunWithRetry` or the caller wrapping terminator
   - `ContinueRun` transition to running
   - `executeContinuation` terminal transitions
   - normal paths only where it reduces duplication without disturbing `RunExecutor.Finalize`
4. Keep `RunExecutor` finalization intact; do not fold it into the new helper unless tests prove equivalent behavior.

Tests:

- `orchestration/terminator_test.go`: successful stop emits status event and run status broadcast.
- `orchestration/continuation_timeout_test.go`: continuation timeout emits one terminal transition and preserves `SessionID`.
- `orchestration/run_actions_test.go`: returned runs/actions update after stop and continue.

### Phase 3 — Durable WebSocket Subscription Intent

Goal: a component can express "I want run X" once and survive socket reconnects.

1. Refactor `useWebSocket` to maintain:
   - `desiredRunSubscriptions: Set<string>`
   - `desiredAllEvents: boolean`
   - `pendingClientMessages` only if needed for non-subscription messages
2. `subscribe(runId)` updates desired state immediately and sends if open.
3. `unsubscribe(runId)` updates desired state immediately and sends if open.
4. `onopen` or `connected` replay sends all desired subscriptions and `subscribe_all` if requested.
5. Remove silent warning-only behavior for closed-socket subscription calls.
6. Add a pure adapter for WS message parsing/building so it is testable without browser WebSocket.

Tests:

- New `ui/tests/lib/webSocketProtocol.test.ts` for proto parse/build behavior.
- New `ui/tests/lib/webSocketSubscriptions.test.ts` for desired subscription replay using a fake socket adapter.
- Existing UI unit tests still pass.

Implementation note: do not add a third-party test framework unless explicitly approved. Prefer pure modules testable by the existing Node test runner.

### Phase 4 — Introduce a Run Event Store

Goal: one frontend state machine owns run realtime coherence.

1. Create pure reducer/store modules, for example:
   - `ui/src/lib/runEventStore.ts`
   - `ui/src/hooks/useRunEventStore.ts`
2. State shape:
   ```ts
   interface RunEventStoreState {
     runsById: Record<string, Run>;
     eventsByRunId: Record<string, RunEvent[]>;
     lastSequenceByRunId: Record<string, bigint>;
     subscribedRunIds: Set<string>;
     allEventsSubscribed: boolean;
     reconnectGeneration: number;
   }
   ```
3. Reducer actions:
   - `runStatusReceived`
   - `runEventReceived`
   - `eventsGapFilled`
   - `runSnapshotLoaded`
   - `taskStatusReceived`
   - `connected`
   - `disconnected`
4. Dedupe by event ID first, then sequence.
5. Gap-fill selected run:
   - On first select: fetch all events.
   - On reconnect: fetch `after_sequence=lastSequence`.
   - On terminal status: fetch `after_sequence=lastSequence`, not a full refetch after `setTimeout(500)`.
6. App-level run list should update from `run_status` patches and bounded periodic/manual refetch, not every `run_event`.
7. Keep `runTimeline.ts` as the display transformer; do not move UI rendering into the store.

Tests:

- `ui/tests/lib/runEventStore.test.ts`:
  - ordered append
  - duplicate event ignored
  - REST gap-fill merges missing events
  - terminal status triggers reconciliation intent
  - reconnect does not drop desired subscriptions
- `ui/tests/lib/runTimeline.test.ts` remains unchanged except for any new event categories.

### Phase 5 — Refactor `App` and `RunsPage` Onto the Store

Goal: remove split-brain UI state.

1. In `App.tsx`:
   - Keep top-level health/profile/task/runs data fetches.
   - Move WS message handling to the run event store.
   - Replace `terminalRunIdsRef` and debounced refetch-on-every-event with store-driven run patches and selective refetch.
2. In `RunsPage.tsx`:
   - Remove local `runOverrides` once store provides snapshots.
   - Remove selected-run `wsAddMessageHandler` logic once store owns selected-run events.
   - Replace `setTimeout(500)` terminal reconciliation with store gap-fill.
   - Keep page concerns: route selection, filters, modals, diff loading, action handlers.
3. In `RunDetail.tsx` and `RunTimeline.tsx`:
   - Keep action rendering driven by `run.actions`.
   - Keep continue/delete-message callbacks, but after mutation update store via returned run/events or explicit refetch.

Tests:

- Add reducer tests first, then minimal component smoke tests only if the existing test stack can support them without new dependencies.
- Manual validation via scenario lifecycle:
  ```bash
  cd scenarios/agent-manager && make start
  cd scenarios/agent-manager && make test
  cd scenarios/agent-manager && make stop
  ```

### Phase 6 — Continue Path Lifecycle Convergence

Goal: reduce duplicated lifecycle code without destabilizing normal runs.

1. Compare `RunExecutor.Execute` and `executeContinuation`.
2. Extract only shared lifecycle primitives at first:
   - status transition helper from Phase 2
   - heartbeat helper with levers
   - transcript preparation helper already present
   - event sink creation helper already partly present as `dispatcherSink` / inline continuation logic
3. Replace continuation inline event sink creation with a shared method.
4. Move continuation timeout defaults into `config.Levers` if any hard-coded timing remains.
5. Document continuation as a separate but aligned temporal flow in `TEMPORAL-FLOWS.md`.

Tests:

- Existing `continuation_timeout_test.go`
- Existing `continue_attachments_test.go`
- New test that protected-mode continuation uses the same launcher path and lifecycle event contract.

### Phase 7 — API/CLI and Interoperability Hardening

Goal: make the contract durable for non-UI clients.

1. Keep proto changes additive. Prefer no proto change for Phase 1-6 unless required.
2. If endpoint responses need hydration, add optional fields rather than changing existing ones:
   - `StopRunResponse.run`
   - `ContinueRunResponse.run` already exists
   - optionally include `last_event_sequence` in run/event responses if not already available from events
3. Ensure `GetRunEvents` `after_sequence` remains the canonical gap-fill API.
4. Check CLI parity:
   - `agent-manager run events` should expose after-sequence/limit if it does not already.
   - `agent-manager run stop` should display final status/run ID in mutation-report style.

Tests:

- Handler tests for response shape.
- CLI command tests if CLI parity changes are made.

### Phase 8 — Documentation Updates

Update after code phases land:

- `docs/internal/SEAMS.md`: add RunEventStore, append-and-broadcast helper, run lifecycle transition helper.
- `docs/internal/TEMPORAL-FLOWS.md`: add WebSocket reconnect/gap-fill flow and continuation lifecycle convergence.
- `docs/internal/COHERENCE_NOTES.md`: record the removal of split App/RunsPage realtime state.
- `docs/internal/UTILS_UNIFICATION_NOTES.md`: record any extracted UI store/protocol utilities.
- `docs/internal/PROBLEMS.md`: add/resolve a technical debt item for realtime split-brain state.
- `api/docs/event-schemas.md`: clarify durable append and sequence semantics.

## 8. Contract Decisions

1. **Durability rule:** persisted run events are the source of truth. WebSocket is a delivery optimization.
2. **Ordering rule:** UI sorts by `sequence`, with timestamp as a tie-breaker only for malformed/legacy data.
3. **Dedupe rule:** event ID wins; sequence is secondary.
4. **Reconnect rule:** selected runs reconcile with REST `after_sequence`, then continue live WS.
5. **Mutation rule:** stop/continue/delete-message must either return enough state to update UI immediately or trigger a targeted store reconciliation.
6. **Action rule:** UI does not infer permissions from status except as a defensive fallback; `run.actions` is canonical.
7. **Compatibility rule:** proto changes must be additive; no enum renumbering or field reuse.

## 9. Testing Plan

Backend:

```bash
cd scenarios/agent-manager/api
go test ./internal/adapters/event
go test ./internal/handlers -run WebSocket
go test ./internal/orchestration -run 'Continue|Stop|RunActions|Terminator'
go test ./internal/orchestration/integration
go test ./...
```

Frontend:

```bash
pnpm --dir scenarios/agent-manager/ui run type-check
pnpm --dir scenarios/agent-manager/ui run test:unit
pnpm --dir scenarios/agent-manager/ui run lint
```

Scenario:

```bash
vrooli scenario test agent-manager
```

Manual validation:

1. Start agent-manager through lifecycle only:
   ```bash
   cd scenarios/agent-manager && make start
   ```
2. Open the UI and select a running run.
3. Disconnect/reconnect browser network or restart the API.
4. Confirm selected timeline gap-fills without duplicate events.
5. Stop a running run and confirm:
   - run list changes to cancelled
   - timeline receives status event
   - action buttons update without manual full refresh
6. Continue a completed run and confirm:
   - user message appears immediately
   - status transitions to running then terminal
   - further continue remains available when session ID is preserved

## 10. Rollout and Validation Checklist

- [x] Phase 1 backend tests pass.
- [x] Phase 2 stop/continue status transition tests pass.
- [x] Phase 3 WebSocket protocol/subscription tests pass.
- [x] Phase 4 store reducer tests pass.
- [ ] Phase 5 UI type-check and unit tests pass.
- [ ] Phase 6 continuation tests pass.
- [ ] `vrooli scenario test agent-manager` passes.
- [ ] Docs updated and references point to live files.
- [ ] No direct scenario execution used; lifecycle commands only.
- [ ] `git diff` reviewed for unrelated changes before handoff.

## 11. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Event append locking slows high-volume runs | Use narrow per-append transactions; benchmark only if tests show meaningful slowdown |
| UI store refactor changes visible behavior | Land pure reducer and tests before wiring components |
| Stop path conflicts with `RunExecutor.Finalize` terminal updates | Keep idempotent transition helper; test double-stop and process-already-gone paths |
| Continue refactor accidentally changes protected sandbox routing | Preserve existing `ResolvedConfig` + `SandboxID` fields and add regression tests |
| Proto response hydration affects clients | Add optional fields only; keep existing fields unchanged |
| Broad App refetch removal hides task/profile changes | Keep explicit task/profile refetch triggers for `task_status` and manual refresh |

## 12. Non-goals and Prohibited Patterns

- Do not introduce Zustand, Redux, RxJS, or another dependency without explicit permission.
- Do not keep parallel event reconciliation paths in `App` and `RunsPage`.
- Do not broadcast durable event types before append success.
- Do not reintroduce status/action inference in UI when `run.actions` is available.
- Do not change runner event payload meanings to suit UI display.
- Do not use direct scenario binaries or `nohup`; use `make start`, `make test`, `make stop`, or `vrooli scenario`.

## 13. Definition of Done

The work is complete when:

- Backend event append is concurrency-safe and append-before-broadcast is enforced.
- Stop and continue status transitions emit consistent status events and WebSocket run status updates.
- `useWebSocket` preserves desired subscriptions across reconnects.
- A single frontend run event store owns live events, run snapshots, sequence tracking, and gap-fill.
- `App` and `RunsPage` no longer maintain competing realtime state machines.
- Existing and new backend/frontend tests pass.
- `vrooli scenario test agent-manager` passes.
- Internal docs describe the new seams and temporal flows.
