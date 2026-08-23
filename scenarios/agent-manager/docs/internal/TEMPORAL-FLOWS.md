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
6. Sandboxed continuation turns call the apply-at-run-end orchestration seam, whose lifecycle policy explicitly selects workspace-sandbox `/turn-checkpoint`; the sandbox returns to `checkpointed` instead of being deleted. Final disposal paths remain separate from turn checkpointing. Checkpoint/apply outcome is recorded in `Run.FinalizationStatus`, not by keeping the runner turn in `running`.

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
turn checkpoint/finalization: finalization_status=succeeded|failed|skipped
```

**Levers:** `Execution.DefaultTimeout` caps each continuation turn. `Heartbeat.RunHeartbeatInterval` controls continuation heartbeats.

**Tests:**
- `internal/orchestration/continuation_timeout_test.go::TestContinuation_HasPerTurnTimeout`
- `internal/orchestration/run_lifecycle_test.go::TestContinueRun_ProtectedSandboxCarriesLauncherInputsAndLifecycleEvents`
- `internal/orchestration/run_lifecycle_test.go::TestContinueRun_EmitsRunningAndTerminalStatusTransitions`
- `internal/domain/run_actions_test.go::TestRunActionsFor_ContinueReason` pins that failed finalization does not block continuation after the runner turn completed.
- `internal/orchestration/phases/finalize_test.go::TestApplyAtRunEnd_FailurePreservesSandbox` pins that finalization failure is recorded without changing completed runner status back to an active state.

## Park/wake (durable wait suspension)

Park/wake is a third launch-related flow, distinct from initial execution and continuation: a *running* run that issues an externally-owned async request (a test-genie suite, a git-control-tower baseline diff) is **parked** — its agent process exits (zero tokens), the run enters the non-terminal `RunStatusParked` with an `AwaitHandle` recorded, and agent-manager performs the blocking wait on the agent's behalf, then **wakes** the run by resuming the conversation with the result injected as the next turn. Park reuses the continuation primitives (`resumeConversation`) for the wake leg, so the env + identity re-injection and heartbeat-reset semantics are shared with `ContinueRun`.

Why a separate flow: a parked run is intentionally process-less and silent, so it must be exempt from the reconciler's heartbeat/process liveness checks (its `LivenessPolicy` is `scanned, no heartbeat/process expectation` — it is listed but never reaped). The park deadline (`DefaultParkTTL`, 30m) is the only bound: when it elapses agent-manager wakes the run with a typed timeout result rather than hanging.

```
runner turn (running) — in-run CLI calls POST /runs/{id}/park (identity-auth: caller must own the run)
   ↓
ParkRunFromAgent → ParkRun: record AwaitHandle, status transition running → parked
   ├─ emit run_status event: "Parked waiting on <producer>:<key> (ETA …)"
   ├─ AwaitRegistry.Register → one background watcher goroutine blocks on the producer's Waiter
   └─ endParkedTurn: after a short grace, terminate the agent process group (turn ends, zero tokens)
        └─ the still-alive executeRun/executeContinuation goroutine re-reads status==parked and
           declines any terminal transition (park owns the lifecycle; sandbox preserved)
   ↓ … parked (durable; survives agent-manager restart via RecoverParkedRuns re-spawning the watcher) …
   ↓
watcher resolves: producer result | producer error | deadline elapsed (timedOut)
   ↓
WakeRun (idempotent — a non-parked run is a no-op): clear AwaitHandle, build typed wake message
   ↓
resumeConversation: status transition parked → running, reset LastHeartbeat=now,
   re-inject custom+sandbox+identity env (fresh identity token), runner.Continue with the result
   ↓
status transition: running → complete|failed (or parked again on the next async request)
```

**Levers:** `DefaultParkTTL` (park deadline, 30m), `DefaultParkTurnEndGrace` (delay before process-group kill, 2s). `Execution.DefaultTimeout`/`Heartbeat.RunHeartbeatInterval` apply to the woken turn (it is a continuation).

**Tests:**
- `internal/orchestration/park_test.go` — `running→parked→running` round-trip preserves identity/env + resets heartbeat; timeout typed result; stop-while-parked → cancelled + handle cleared; wake idempotent; double-park rejected.
- `internal/orchestration/park_from_agent_test.go` — identity-auth (owning token parks; foreign token → 403; invalid token; already-parked rejected).
- `internal/orchestration/await_registry_test.go` — resolve/producer-error/deadline-timeout/cancel-no-wake/restart-recovery/idempotent-register.
- `internal/database` `TestTouchHeartbeat_StatusGuarded` — a heartbeat tick during the park window cannot resurrect `parked→running` (atomic status-guarded update).

## Workflow completion nudge + blocking wait

Workflow executions advance through a **crash-safe pull loop** (`driveWorkflowExecution`): it reads durable state, calls the engine's `Advance`, and stops at the no-progress version fixpoint when a dispatched run is still non-terminal. Nothing re-drives the execution on its own — so before Phase 4 consumers polled `AdvanceWorkflowExecution` on a ticker to notice a run had finished.

Two mechanisms replace that poll, neither a new scheduler:

**Completion nudge (push the pull loop).** When a run belonging to a workflow attempt reaches terminal (the run-terminal transition point in `executeRun`/`resumeRun`/`executeContinuation`), the orchestrator resolves the owning execution via `WorkflowExecutionRepository.ExecutionIDForRun` (the reverse of the one-directional `attempt.RunID` link) and enqueues an idempotent drive on the in-process `WorkflowNudger`. A child-workflow execution settling terminal likewise nudges its `ParentExecutionID`. The drive re-reads durable state and is guarded by the engine's optimistic-version CAS, so a nudge racing an explicit `Advance` (or a second nudge) is safe — the loser rereads and exits at the fixpoint. The nudge queue **dedupes** (one pending drive per execution) and clears the pending mark before driving so a completion landing mid-drive re-queues.

```
run terminal (executeRun returns, not parked)
   ↓
ExecutionIDForRun(runID) ──▶ uuid.Nil? ── yes ─▶ ignore (non-workflow run)
   ↓ no
WorkflowNudger.Enqueue(executionID)  (dedupe, non-blocking)
   ↓
worker: driveWorkflowExecution(executionID)  (CAS-guarded; concurrent-safe)
   ↓
execution settles terminal ──▶ notify blocking waiters + nudge parent
```

**Blocking wait (server-owned long-poll).** `WaitWorkflowExecution(executionId, timeout)` blocks until the execution is terminal or the deadline passes, mirroring test-genie's `WaitRun`. It is event-driven — it subscribes to a per-execution notifier, re-reads durable state to close the subscribe/settle race, then blocks on a wake channel; there is **no ticker and no per-wait poller**. `timeout <= 0` blocks until terminal (or the caller's context is cancelled). **Cancelling the waiter never cancels the execution** — the wait only reads execution state; the execution is driven independently by the engine, the nudge, and the reconciler backstop. This is the documented adoption pattern: adopters call `StartWorkflowExecution` then `WaitWorkflowExecution`, and write no poller.

**Durable backstop.** A crash between the run terminal and the enqueue is covered by `RecoverWorkflowExecutions` (the reconciler recovery sweep, at boot and every reconcile cycle), which re-drives every non-terminal recoverable execution. The nudge is an optimization over this durable path, never the only route to progress.

**`AdvanceWorkflowExecution` is ops-only.** With the nudge and wait in place, no consumer calls `AdvanceWorkflowExecution` in the normal flow. It remains as a manual/operator recovery verb — force-reconcile one execution from its durable state (e.g. after a stuck child, or to inspect drive behavior). It is idempotent and CAS-guarded, so an operator advance racing a nudge is safe.

**Levers:** `Workflow.NudgeWorkers` (concurrent drive workers, default 4), `Workflow.NudgeDriveTimeout` (per-nudge drive bound, default 5m). The wait timeout is per-request. See [`../reference/configuration.md`](../reference/configuration.md#workflow).

**Tests:**
- `internal/orchestration/workflow_nudge_test.go` — nudger dedupe, concurrent drain (-race), wait-registry notify.
- `internal/orchestration/workflow_wait_test.go` — immediate-terminal return, block-until-terminal, deadline→timed_out, concurrent waiters, cancel-does-not-cancel-execution.
- `internal/orchestration/workflow_nudge_integration_test.go` — run node reaches its next node with zero `AdvanceWorkflowExecution` calls; nudge tolerates concurrent explicit advance; progression across a simulated restart via the recovery backstop.
- `internal/database/repository_workflow_test.go::TestWorkflowExecutionRepositoryExecutionIDForRun` — the reverse run→execution link.

## Lifecycle ownership and operator diagnostics

The durable lifecycle is intentionally split at the wait boundary:

| State | Durable owner | Process expectation | Recovery action |
| --- | --- | --- | --- |
| `running` | Run executor | one runner turn may be active | inspect heartbeat and runner process |
| `parked` | Await registry + run repository | no runner process is required | recover the persisted await handle and reattach one watcher |
| `running` after wake | Run executor | one continuation turn may be active | verify the typed wake result was injected |
| terminal | Run repository / workflow repository | no runner process | reconcile dependents from the durable terminal record |

The parked record is the source of truth. A watcher never creates a replacement
operation and a process liveness scan never reaps a parked run. The handle
(`producer`, `key`, `deadline`, registration time), last heartbeat, run id, and
workflow id are the minimum evidence needed to diagnose a recovery:

```text
parked run → producer/key + deadline + last heartbeat
          → startup reattach (one watcher)
          → producer result | typed timeout | typed producer failure
          → clear handle before wake
          → continuation or terminal failure
```

Operator guidance: if a run remains parked, report the run id, producer/key,
deadline, last heartbeat, and the next action (`await`, `timeout`, `cancel`, or
`inspect producer`). Do not manually start a second wait. A human shell wait
may remain blocking in the human session, but managed agents must use the
server-owned park/wake contract so restart recovery and cancellation remain
observable.

### Recovery runbook

1. Read the run and confirm `status=parked`, `await_handle.producer`,
   `await_handle.key`, `deadline`, and `last_heartbeat`.
2. Check the producer's existing operation by that same key. Do not submit a
   replacement operation.
3. If the producer is terminal, allow startup recovery or an explicit reattach
   to invoke the same waiter; it will wake the run once.
4. If the deadline has elapsed, expect the typed timeout wake. If the producer
   failed, expect a typed producer-error wake. If the run is intentionally
   stopped, expect cancellation and no continuation.
5. After wake, verify the handle is cleared, `last_await_result` is present,
   heartbeat is fresh, and the continuation has the same conversation,
   sandbox, working directory, custom environment, and a valid fresh identity.

### Troubleshooting ownership failures

- Parked with no watcher: inspect Agent Manager startup recovery and the
  persisted handle; do not restart the producer.
- Repeated wake attempts: inspect run events and `last_await_key`; a
  non-parked run is an idempotent no-op.
- Stale heartbeat while parked: expected; parked runs are exempt from runner
  liveness reaping. Check the deadline and producer state instead.
- Missing continuation environment or identity: treat as a lifecycle defect;
  the wake path must rebuild it, not ask the agent to reconstruct private state.

## Post-run sandbox finalization

Runner turn status and sandbox finalization are separate temporal flows. A runner can finish and emit several assistant messages before process exit; assistant messages are not terminal. The terminal signal is the runner result/process completion path. After that, sandbox apply/checkpoint runs as post-turn finalization and records one of:

- `none` / `skipped`: no sandbox finalization was needed or policy deferred it.
- `running`: finalization is currently applying/checkpointing.
- `succeeded`: apply/checkpoint finished.
- `failed`: apply/checkpoint failed after the runner turn completed.

`CanContinueRun` gates on runner turn activity and session availability, not on finalization success. A run with `status=complete`, a session id, and `finalization_status=failed` can be continued while the UI shows the structured finalization warning.

Workspace-sandbox availability during finalization is recovered only for retryable transport failures:

```
ApplyAtRunEnd / TurnCheckpoint
   ↓
workspace-sandbox transport failure before response
   ↓
WorkspaceSandboxEnsurer.EnsureAvailable(ctx)
   ↓
retry with Sandbox.OperationMaxAttempts + bounded backoff
   ↓
finalization_status=succeeded|failed
```

The finalization context remains detached from the runner execution context and bounded by `Heartbeat.TeardownTimeout`; the sandbox retry levers cannot extend finalization indefinitely.

## Sandboxed setup availability

Fresh sandboxed run setup performs a run-time dependency check before creating a new sandbox:

```
SetupWorkspace(new sandbox)
   ↓
provider.IsAvailable(ctx with Sandbox.AvailabilityCheckTimeout)
   ├─ healthy → CreateSandboxWorkspace
   └─ unavailable
        ↓
      WorkspaceSandboxEnsurer.EnsureAvailable(ctx)
        ├─ lifecycle start: vrooli --no-stale-check scenario start workspace-sandbox
        └─ health poll until Sandbox.EnsureStartTimeout
   ↓
Create(idempotencyKey=sandbox:run:{runID})
   └─ retry retryable create failures with same idempotency key
```

Existing sandbox reuse and in-place runs skip this dependency ensure path. Agent-manager process bootstrap also skips it; startup-time dependency management remains owned by Vrooli lifecycle declarations.

**Levers:** `Sandbox.AvailabilityCheckTimeout`, `Sandbox.EnsureStartTimeout`, `Sandbox.EnsurePollInterval`, `Sandbox.OperationMaxAttempts`, `Sandbox.OperationInitialBackoff`, `Sandbox.OperationMaxBackoff`.

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

## Workflow-execution timeline

The workflow runtime extends the existing detached Run and durable park/wake
mechanisms. The implemented ordering is:

```text
start idempotency reservation
  -> pin workflow definition digest and typed input snapshot
  -> append execution-created journal entry
  -> choose runnable nodes from graph + join + budget state
  -> persist node-attempt dispatch intent and child idempotency identity
  -> dispatch fresh Run / named continuation / child workflow, or enter wait
  -> persist canonical child result and budget consumption
  -> append node-completed journal entry
  -> evaluate declared branch/edge and increment traversal counter
  -> repeat, wait, or persist one typed terminal outcome
```

Recovery may occur between any two arrows. It reads the persisted intent before
performing a side effect and the persisted completion before advancing an edge.
Revisiting a `run` node creates a new Run identity; recovery of an already
dispatched attempt reattaches to that same Run. A `continue` node carries a
separate idempotency identity and must reference an earlier conversation in the
same execution. Canonical RunResult selection first scopes provider evidence to
the latest durable `turnId`, so terminal outputs from earlier continuation
turns cannot make the current handoff ambiguous after recovery.

The append-only journal is the only cross-node context source. A later prompt
receives only explicitly selected, ordered entries rendered within declared
byte/item/depth limits. No prior transcript, tool event, or mutable variable bag
is inherited implicitly. Each wait visit persists a distinct correlation key,
resume token, and deadline and resumes only through a validated signal recorded
after that visit's intent or a timeout transition.

Every cycle-forming edge has a positive per-edge traversal cap. Traversals,
retries, child runs, subworkflows, continuations, and waits also consume global
time/turn/token/cost/attempt/child/concurrency/recursion budgets. Exhaustion is a
typed `budget_exhausted` terminal, never an unbounded retry or silent failure.

### Durable core interpreter commit sequence

1. Start validates the input against the pinned revision and atomically creates
   `WorkflowExecution(version=1)` plus journal sequence 1.
2. At a run or continue node, the interpreter renders only declared bounded
   bindings, then atomically commits a `dispatch_pending` attempt containing
   immutable input and prompt snapshots. No agent side effect precedes this.
3. The canonical Run creation or continuation service receives the persisted
   attempt idempotency key. A crash/retry therefore returns the same Run or
   continuation instead of sending twice. Node-local turn and timeout limits
   are included in that request; continuations apply them to the resumed turn
   without mutating the source Run's persisted resolved configuration.
4. A second compare-and-swap commit attaches Run and conversation identity.
   Waiting consumes no agent turn. The reconciler inspects recoverable
   executions on boot and every normal reconciliation cycle.
5. Terminal child evidence, canonical RunResult, structured result, compact
   handoff, attempt completion, budget usage, edge traversal, and next-node
   pointer commit atomically. Recovery can never observe a completed attempt
   without its journal evidence or an edge advance without its counter.

All execution updates require the previously read version. A competing API or
reconciler advance loses the compare-and-swap and rereads; it cannot duplicate
the child side effect because the attempt intent and child idempotency key are
already durable.

After each successful journal commit, the orchestration layer emits a
metadata-only WebSocket lifecycle projection. It includes correlation ids,
node strategy/profile, child Run and conversation ancestry, journal sequence
and digest, counters, budgets, and terminal reason. Prompt snapshots and result
bodies are never broadcast. Operators can recover the same ordered metadata
from `GET /api/v1/workflow-executions/{id}/trace` after reconnect or restart.

### Wait, signal, composition, and cancellation sequence

Wait nodes commit a workflow-owned correlation key, opaque resume token,
payload schema, and bounded deadline before entering `waiting`. They do not
park a Run and consume no agent turn. A signal validates execution identity,
signal name, payload schema, idempotency key, and optional expected execution
version before its journal commit. Duplicate signals return the committed
state; late, wrong-type, and wrong-execution signals are typed conflicts. Boot
reconciliation revisits waiting executions to observe a committed signal or a
deadline, without keeping a goroutine alive per wait node. This is distinct from
the request-scoped `WaitWorkflowExecution` long-poll (see "Workflow completion
nudge + blocking wait"), which holds one select on a wake channel for the life
of the client call and observes terminal state — it neither polls nor drives the
execution, and cancelling it leaves the execution untouched.

Child-workflow attempts commit input and dispatch identity before creating a
digest-pinned child `WorkflowExecution`. The child records parent execution and
attempt IDs plus recursion depth. Recovery reuses the attempt idempotency key
and child ID; completion aggregates the child's budget ledger into the parent
tree before the parent edge advances.

A parallel branch atomically commits a durable fork-visit start marker, its
complete membership, and every member dispatch intent. Members may be fresh
Runs, named continuations, or child workflows and are dispatched only up to the
workflow concurrency bound; wider fan-out stays durably dispatch-pending and is
released in later batches. A completed-visit marker is committed with the
move to the join. Returning through a cycle therefore increments the visit and
creates new per-node ordinals and idempotency keys, while crash recovery within
an unfinished visit reuses the same attempts. All members converge on one
`all`, `any`, or positive `quorum` join; `any` and `quorum` advance immediately
when satisfied, fail when success becomes impossible, and stop/mark remaining
losers before the join commit. Their child identities and completion evidence
remain separate in the journal.

Cancellation first commits non-terminal `cancelling`, recursively propagates to
active Runs and child workflows, then records a cleanup disposition and commits
`cancelled`. Failure and budget-exhaustion paths also require a cleanup
disposition. Recovery includes cancelling rows and abnormal terminal rows whose
current retry generation has no cleanup entry, closing crash windows between
parent state and child stop.

## Adding a new temporal flow

Following the architectural invariant 4 (Levers is the only home for adjustable thresholds):

1. Pick the section in `config.Levers` that owns this flow. If none fits, add a new sub-struct.
2. Document the lever in `docs/reference/configuration.md`.
3. Add the cadence entry to this file (timeline + lever + test).
4. Pin the cadence with a unit test that uses a fast override.

If you find yourself adding a `time.Sleep(X)` literal or a `time.NewTicker(X)` literal anywhere in agent-manager source, stop — promote `X` to a lever first.
