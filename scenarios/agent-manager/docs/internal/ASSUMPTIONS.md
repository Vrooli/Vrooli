# Assumptions

Things the code assumes about the world. Distinct from [`INVARIANTS.md`](INVARIANTS.md) — invariants we *enforce*, assumptions we *depend on*. Assumptions can stop holding (a third party changes its protocol; a runtime resource changes shape) and the code will quietly mis-behave; documenting them makes the failure mode visible when an audit notices the assumption no longer applies.

When an assumption stops holding, either:

1. The code learns to handle the new world (and the assumption is updated or removed), or
2. The assumption is promoted to an invariant (with a test that pins the world the way the code wants it).

## A1. Codex's per-thread state lives at `~/.codex/state_5.sqlite`

Codex CLI persists thread metadata, conversation state, and the in-progress rollout writer pointer in a single SQLite database under `$CODEX_HOME/state_5.sqlite` (current as of codex 0.5.x). The bootstrap window — between `exec.CommandContext` returning and the first JSON event landing on stdout — opens this DB, acquires its WAL lock, registers an in-memory rollout writer, and opens the rollout file.

Concurrent codex processes against the same `$CODEX_HOME` race steps 1–3. The `spawn.Dispatcher` (Invariant I2) was added because of this assumption. If codex's state-DB filename or schema version changes, the dispatcher's purpose remains correct but error-classification fixtures (`codex.go::ClassifyTerminalError`) may need new patterns.

**How to detect drift:** Watch for a new `record_rollout_items` failure pattern that doesn't match either the `_STATE_LOST` or `_EXPIRED` shape. That's a signal codex has changed its bootstrap sequence.

## A2. The workspace-sandbox merged dir is exposed at `/workspace` inside the sandbox namespace

`SandboxLauncher.translateHostPathToNamespace` rewrites `LaunchRequest.WorkingDir` (and any matching env values) from the host's merged-dir path to `/workspace` before posting to `/api/v1/sandboxes/{id}/processes`. The constant must stay aligned with the `--bind <merged> /workspace` arg in `workspace-sandbox/api/internal/driver/bwrap.go`.

If workspace-sandbox renames or reshapes the bind mount, `SandboxNamespacePath` in `internal/adapters/sandbox/sandbox_launcher.go` must change in lockstep. There is currently no automated cross-scenario contract test (it would require a running workspace-sandbox); regression manifests as `*LaunchBlocked{Code:"workdir_outside_sandbox"}` for runs that previously worked.

## A3. Codex's `cwd` reaches the rollout file as the post-translation namespace path

For a sandboxed run, the codex process inside the bwrap namespace observes `pwd == /workspace`, which is what lands in `~/.codex/sessions/.../rollout-*.jsonl`. This is the audit-trail signal that downstream consumers (most prominently Git Control Tower) depend on to attribute changes to the correct sandbox. The `sandbox_cwd_contract_test.go` integration test pins this — if it fails, sandbox bypass has reappeared in some form.

## A4. Heartbeat-driven callers may fire `CreateRun` bursts on shared seconds

prompt-manager team agents and swarm-manager initiative orchestrators run on periodic ticks. When several ticks align (most commonly on minute boundaries, when multiple cron-like schedulers fire together), N `CreateRun` calls land at agent-manager within a few ms of each other. Invariant I2 (dispatcher serializes startup) handles this; this assumption documents *why* the dispatcher exists rather than relying on callers to back off.

If callers gain coordinated scheduling (i.e. the heartbeat schedulers learn about each other), the dispatcher's purpose narrows — but the failure mode it prevents (codex bootstrap-window contention) is still possible from any unrelated burst.

## A5. The `obs.Logger` writer survives across a single `obs.Init` call per process

`obs.Logger()` returns the package-level `*slog.Logger` installed by `obs.Init` at server startup. Tests that need to capture log output use `obs.InitWithWriter` (a single-call swap; the previous logger is replaced atomically). Production runs call `obs.Init` exactly once.

If a test runs `obs.InitWithWriter` then forgets to restore the original logger, a later test may fail to see expected output. Tests that assert on log content should always reset via `t.Cleanup`. This isn't enforced — it's an assumption about test hygiene.

## A6. `CreateRunResponse.queue_depth` is observed by callers that care about backpressure

The dispatcher's queue depth is on the response, not on a side-channel; callers can decide to skip a tick when the queue is non-trivially deep. We assume that *some* heartbeat-driven callers will choose to act on this signal eventually (the work is on the prompt-manager / swarm-manager side, not agent-manager's). Until they do, the field's only consumer is operator-facing UI.

If queue depths consistently sit above zero in production with no caller adjusting, that's a signal the dispatcher levers (`MaxStartingConcurrency`, `QueueCapacity`) need raising — not that the assumption is wrong.

## A7. SQLite's WAL mode tolerates the orchestrator's connection-pool size

`Storage.MaxOpenConns = 25` (default) feeds a single SQLite database in WAL mode. SQLite's WAL handles many readers and a single writer; the orchestrator's writer concurrency is bounded by the `runs` and `events` tables' single-writer semantics, not by the pool size. We assume that 25 is enough headroom for read fan-out (the UI's run-list, the WebSocket transcript replay, and the reconciler all read concurrently).

If an operator sees `database is locked` errors at steady state, the pool is the wrong knob — the writer needs serialization, not more readers.

## How to add an assumption here

1. Frame as something the code *takes for granted about the world*, not something the code itself enforces.
2. Note how drift would manifest. "If X changes, Y will start failing with shape Z" is the high-value framing.
3. If the assumption is structural (e.g. a constant must match a constant in another scenario), cross-link both files.

## Related

- [`INVARIANTS.md`](INVARIANTS.md) — statements the code enforces.
- [`SEAMS.md`](SEAMS.md) — testability boundaries.
- [`TEMPORAL-FLOWS.md`](TEMPORAL-FLOWS.md) — cadences and timing.
