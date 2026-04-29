# Temporal Flows

Every cadence, timeout, and timing-sensitive contract in agent-manager lives in `config.Levers`. This document maps the timing surfaces to the levers that control them and the test that pins them.

## Run execution timeline

```
   t=0     submit run
   t=0+    Setup phase (sandbox create) ......... bounded by Execution.DefaultTimeout
   t=…     Acquire phase (runner.IsAvailable) ... bounded by Runners.ProbeTimeout
   t=…     Generate identity token .............. unbounded (HMAC sign)
   t=…     Execute phase
            • runner Execute() blocks ........... cancelled at Execution.DefaultTimeout
            • heartbeat goroutine ............... fires every Heartbeat.RunHeartbeatInterval
            • checkpoint persistence ............ every Heartbeat.CheckpointInterval
   t=end   HandleResult (classify + apply)
   t=end+  Finalize (DEFERRED) ................. bounded by Heartbeat.TeardownTimeout
            • detached from execCtx — runs
              even after deadline exceeded
```

**Lever:** `Execution.DefaultTimeout` (default 30m) caps the entire pipeline through HandleResult. Finalize uses `Heartbeat.TeardownTimeout` so sandbox teardown completes even after the run timed out.

**Test:** `phases/finalize_test.go::TestApplySandboxLifecycle_DeletesEvenWithCancelledCallerCtx` pins the 2026-04-28 mount-leak fix.

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
