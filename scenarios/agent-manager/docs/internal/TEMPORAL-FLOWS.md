# Temporal Flows

Every cadence, timeout, and timing-sensitive contract in agent-manager lives in `config.Levers`. This document maps the timing surfaces to the levers that control them and the test that pins them.

## Run execution timeline

```
   t=0      submit run (CreateRun)
   t=0      dispatcher.Enqueue ................. bounded by Spawn.QueueCapacity
   t=q      slot acquired (queue → starting) ... bounded by Spawn.MaxStartingConcurrency
   t=q+     optional MinSpacing delay .......... Spawn.MinSpacing
   t=s      executeRun begins
            • Setup phase (sandbox create) ..... bounded by Execution.DefaultTimeout
            • Acquire phase (runner.IsAvailable) bounded by Runners.ProbeTimeout
            • Generate identity token .......... unbounded (HMAC sign)
            • Execute phase
              ─ runner Execute() blocks ........ cancelled at Execution.DefaultTimeout
              ─ heartbeat goroutine ............ fires every Heartbeat.RunHeartbeatInterval
              ─ checkpoint persistence ......... every Heartbeat.CheckpointInterval
              ─ Started() callback fires once
                Status = RunStatusRunning ...... releases the dispatcher slot
   t=end    HandleResult (classify + apply)
   t=end+   Finalize (DEFERRED) ................ bounded by Heartbeat.TeardownTimeout
            • detached from execCtx — runs
              even after deadline exceeded
```

**Lever:** `Execution.DefaultTimeout` (default 30m) caps the entire pipeline through HandleResult. Finalize uses `Heartbeat.TeardownTimeout` so sandbox teardown completes even after the run timed out. Startup serialization is bounded by `Spawn.MaxStartingConcurrency` and `Spawn.MinSpacing`.

**Test:** `phases/finalize_test.go::TestApplySandboxLifecycle_DeletesEvenWithCancelledCallerCtx` pins the 2026-04-28 mount-leak fix.

## Spawn-startup window

The codex bootstrap window — the interval between `exec.CommandContext` returning and codex emitting its first JSON event — is the most fragile interval in the runner pipeline. During this window codex:

1. Opens `~/.codex/state_5.sqlite` and acquires the WAL lock.
2. Registers the in-memory rollout-items writer task.
3. Opens `~/.codex/sessions/.../rollout-*.jsonl` for append.
4. Emits `session_meta` and the initial `turn_context` events.

Steps 1–3 are not safe to overlap with another codex process starting at the same time on the same `$CODEX_HOME`. Heartbeat-driven callers — most prominently prompt-manager team agents firing on a shared tick — used to spawn N codex processes at the exact same UTC second, racing this window. The visible failure mode was multiple concurrent rollout files all stalling within ~3s with only initial events written.

The `spawn.Dispatcher` shrinks this window's concurrency to `Spawn.MaxStartingConcurrency` (default 1). The slot is held until either the executor calls the injected `Started()` callback (signalling `RunStatusRunning`) or `ExecuteFn` returns — whichever fires first. The defer-release semantics inside the dispatcher worker protect against panics and early-exit terminal failures.

```
heartbeat tick → CreateRun #1 ┐
heartbeat tick → CreateRun #2 ├─▶ dispatcher.Enqueue (FIFO)
heartbeat tick → CreateRun #3 ┘
                                 │
                                 ▼
                          worker goroutine
                                 │
                          ┌──────┴──────┐
                          │  semaphore  │  cap = MaxStartingConcurrency
                          └──────┬──────┘
                                 │
                                 ▼
                       acquire slot → optional MinSpacing
                                 │
                                 ▼
                            ExecuteFn(run, started)
                                 │
                       Started() releases slot
                       (or defer-release on exit)
```

**Levers:** `Spawn.MaxStartingConcurrency`, `Spawn.MinSpacing`, `Spawn.QueueCapacity`. See [`../reference/configuration.md`](../reference/configuration.md#spawn).

**Tests:**
- `internal/orchestration/spawn/dispatcher_test.go` — unit: capacity, panic safety, idempotent `Started()`, `MinSpacing` enforcement.
- `internal/orchestration/integration/spawn_serialization_test.go` — burst gate: 10 concurrent CreateRun calls must keep `in Execute` ≤ `MaxStartingConcurrency` at every observation.

## Heartbeat-driven caller burst pattern

Multiple Vrooli scenarios drive agent-manager from a periodic heartbeat (most notably prompt-manager team agents and swarm-manager initiative orchestrators). When several heartbeats fire on the same second, the resulting `CreateRun` calls land at agent-manager within a few ms of each other.

Symptoms before the dispatcher: silent rollout-file truncation; `record_rollout_items: thread … not found` errors that surfaced as bare `code: INTERNAL`; runs that completed successfully on retry but never explained the first-attempt failure.

Mitigation:

- `spawn.Dispatcher` serializes the bootstrap window even when the heartbeat fires N requests on the same tick.
- `Codec.ClassifyTerminalError` maps the rollout-writer race to typed `ErrCodeRunnerSessionStateLost` so the first-attempt failure is still legible if it does occur.
- Callers can still detect backpressure: `CreateRunResponse.queue_depth` is non-zero when serialization is in effect, so heartbeat-driven schedulers can choose to skip a tick rather than stack work behind a deep queue.

## Heartbeat cadence

The heartbeat goroutine is started at the top of `RunExecutor.Execute()` and runs until the deferred `stopHeartbeat()` fires. Its sole purpose is to update `Run.LastHeartbeat` so the reconciler can detect stalled runs.

```
goroutine starts → SendHeartbeat (immediate)
                    │
                    ▼
              ticker.NewTicker(RunHeartbeatInterval)
                    │
                    ▼
              for { select { case <-ticker.C: SendHeartbeat() ... } }
```

**Lever:** `Heartbeat.RunHeartbeatInterval` (default 15s). The reconciler treats a run as stale when `time.Since(LastHeartbeat) > Concurrency.StaleThreshold` (5m default).

**Test:** `internal/orchestration/run_executor_test.go::TestRunExecutor_Execute_*` exercises heartbeat-on-execute. Reconciler tests pin the staleness detection.

## Continuation turn lifecycle

`ContinueRun` is a separate temporal flow from initial execution because it resumes an existing runner session instead of recreating the workspace and acquiring a fresh sandbox. It is aligned with `RunExecutor` through shared primitives:

1. `ContinueRun` prepares the existing sandbox before mutating run status: `checkpointed` sandboxes are resumed through `sandbox.Provider.Resume`, `active` sandboxes proceed, and terminal/error sandboxes fail without falling back to host execution.
2. `applyRunStatusTransition` moves the run from terminal status to `running`, persists the run, appends a durable status event, and broadcasts hydrated actions.
3. `runEventSink` creates the same durable event-sink shape used by dispatcher lifecycle events: append-and-broadcast when both store and broadcaster exist, append-only when only the store exists, and no-op when event storage is absent.
4. `executeContinuation` derives `Execution.DefaultTimeout` and `Heartbeat.RunHeartbeatInterval` from the same lever set as `RunExecutor`.
5. `phases.RunHeartbeatLoop` sends continuation heartbeats until the runner returns, then the terminal `applyRunStatusTransition` records the completed or failed status.
6. Sandboxed continuation turns call the apply-at-run-end orchestration seam, whose lifecycle policy explicitly selects workspace-sandbox `/turn-checkpoint`; the sandbox returns to `checkpointed` instead of being deleted. Final disposal paths remain separate from turn checkpointing.

```
ContinueRun
   ↓
sandbox prepare: checkpointed → active via Resume
   ↓
status transition: terminal → running
   ↓
emit user message event
   ↓
runner.Continue(ctx with Execution.DefaultTimeout)
   ├─ heartbeat loop @ Heartbeat.RunHeartbeatInterval
   ├─ runner EventSink.Emit → durable event stream
   └─ ContinueRequest carries ResolvedConfig + SandboxID
   ↓
status transition: running → complete|failed
   ↓
turn checkpoint: active → checkpointed
```

**Levers:** `Execution.DefaultTimeout` caps each continuation turn. `Heartbeat.RunHeartbeatInterval` controls continuation heartbeats.

**Tests:**
- `internal/orchestration/continuation_timeout_test.go::TestContinuation_HasPerTurnTimeout`
- `internal/orchestration/run_lifecycle_test.go::TestContinueRun_ProtectedSandboxCarriesLauncherInputsAndLifecycleEvents`
- `internal/orchestration/run_lifecycle_test.go::TestContinueRun_EmitsRunningAndTerminalStatusTransitions`

## WebSocket reconnect and gap-fill

The UI treats WebSocket as a delivery optimization over the durable run-event store. The selected run's last observed sequence is the resume cursor.

```
App mounts
   ↓
useWebSocket connects
   ↓
subscription intent replay
   ├─ subscribe_all if requested
   └─ subscribe run_id for every desired selected-run subscription
   ↓
live messages
   ├─ run_status → RunEventStore run snapshot
   ├─ run_event  → RunEventStore ordered event append + dedupe
   └─ task_status → task refresh path

socket disconnect/reconnect
   ↓
RunEventStore records reconnect generation
   ↓
selected run emits reconciliation intent
   ↓
GET /api/v1/runs/{id}/events?after_sequence=<lastSequence>
   ↓
eventsGapFilled merges missing durable events without duplicates
```

Terminal run statuses use the same `after_sequence` reconciliation path; fixed sleeps are not part of the contract. `GetRunEvents.after_sequence` returns events with `sequence > after_sequence`, so clients can safely pass their last displayed sequence.

**Tests:**
- `ui/tests/lib/webSocketSubscriptions.test.ts` pins subscription intent replay.
- `ui/tests/lib/runEventStore.test.ts` pins ordered append, dedupe, reconnect intent, and gap-fill merge behavior.
- `api/internal/handlers/handlers_test.go::TestGetRunEvents_Success` pins the REST `after_sequence` filter.

## Recovery tail cadence

When a run is recovered with `allowTail=true` and the runner process is alive, `Reconciler.startTailer` opens a goroutine that polls the transcript file every `Recovery.TranscriptTailInterval` and forwards new lines through the recovery EventSink.

```
RecoverInFlightRuns
   ↓
recoverRun (allowTail=true)
   ↓ no terminal in transcript yet
isProcessAlive?
   ↓ yes
startTailer goroutine ──▶ runner.Consume (poll @ TranscriptTailInterval)
                                ↓
                             EventSink.Emit (through Gate when wired)
```

**Lever:** `Recovery.TranscriptTailInterval` (default 100ms). Lower = faster tail catch-up, more file-system reads.

**Tests:**
- `internal/orchestration/recovery_test.go` — unit tests for individual recovery primitives.
- `internal/orchestration/integration/restart_resume_test.go::TestRestartResume_TranscriptReplayCompletes` — end-to-end regression gate.

## Sandbox SSE consumption

When the run is sandboxed, `SandboxLauncher` consumes a Server-Sent Events stream from `workspace-sandbox` to learn the agent process exit status. The SSE consumer in `sandbox_launcher.go` blocks on `LogStream` until an exit event arrives or the connection drops.

The 2026-04-28 SSE Flusher bug (workspace-sandbox `responseWriter` middleware not implementing `http.Flusher`) broke every sandboxed run by returning HTTP 500 from `/processes/{pid}/logs/stream`. The fix is in `scenarios/workspace-sandbox/api/main.go:394-405`. Treat any "sandbox process ended without exit info" / `INTERNAL` event in production as a regression of this bug.

**Lever:** `Heartbeat.RunnerSignalGracePeriod` controls the SIGTERM→SIGKILL window for forced runner shutdown.

## Finalize-deferred lifetime

`phases.Finalize` is registered as `defer` at the top of `RunExecutor.Execute()`. It always runs, including on panic and ctx cancellation. The `finalized` flag on `RunExecutor` makes re-entry a no-op.

Internally Finalize:
1. Advances the phase ladder to `RunPhaseCleaningUp`.
2. Calls `ApplySandboxLifecycle` (Delete or Stop based on lifecycle config).
3. Advances to `RunPhaseCompleted`.
4. Persists final run state and broadcasts terminal status.

All of this happens under a fresh `context.Background()`-derived context with `Heartbeat.TeardownTimeout` so the supplied caller ctx (which may already be cancelled) does not abort teardown.

**Lever:** `Heartbeat.TeardownTimeout` (default 30s).

**Test:** `phases/finalize_test.go` covers all paths — success, failure, cancel, partial apply, manual review, transport failure, in-place no-op.

## Idle-detection windows

Runner-internal idle detection (the moment a runner concludes the agent has gone silent) is controlled by `Heartbeat.AgentIdleThreshold` and per-runner sub-fields. These are codec-internal and surface to the user as "idle warning" events.

**Lever:** `Heartbeat.AgentIdleThreshold` (per-runner, see codecs).

## Test cadence vs production cadence

Tests use `config.DefaultLevers()` with selective overrides that shrink intervals to milliseconds so unit tests don't take hours. Examples:

```go
levers := config.DefaultLevers()
levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond
levers.Recovery.TranscriptTailInterval = 5 * time.Millisecond
executor.WithLevers(levers)
```

Production defaults stay realistic; only tests override.

## Adding a new temporal flow

Following the architectural invariant 4 (Levers is the only home for adjustable thresholds):

1. Pick the section in `config.Levers` that owns this flow. If none fits, add a new sub-struct.
2. Document the lever in `docs/reference/configuration.md`.
3. Add the cadence entry to this file (timeline + lever + test).
4. Pin the cadence with a unit test that uses a fast override.

If you find yourself adding a `time.Sleep(X)` literal or a `time.NewTicker(X)` literal anywhere in agent-manager source, stop — promote `X` to a lever first.
