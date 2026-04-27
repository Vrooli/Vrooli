# Runner Transcript Recovery Implementation Plan

## Purpose

Make agent-manager runs survive an agent-manager process restart. When agent-manager dies (planned restart, crash, or `vrooli scenario restart agent-manager` invoked from inside its own run), the still-running runner subprocess and any events it emits afterwards must be recoverable: agent-manager comes back up, finds the runs that were in flight, drains their durable transcript file, and either re-attaches to a still-living runner or finalizes the run from the terminal events on disk.

This unifies across all three runners (`claude-code`, `codex`, `opencode`) by making agent-manager — not the runner CLI — own the durable record of every line a runner emitted.

## Hard Rule: No Self-Restart Guard

**Do NOT add a CLI guard that prevents `vrooli scenario restart agent-manager` from being invoked while an agent-manager-managed run is in progress.** Restarting agent-manager from inside a run is a legitimate workflow when the change being applied is to agent-manager itself. The fix here is to make agent-manager survive that restart cleanly, not to forbid the operation. Repeat in Definition of Done.

## Required Reading

Before executing this plan, the implementing agent must run:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

These cover the seam-discipline and unification patterns that govern how the three runner adapters share infrastructure, and the CLI/API conventions for any new endpoints.

## Problem Statement

When agent-manager receives SIGTERM (graceful restart) or SIGKILL (crash), in-flight runs are lost in a way that loses already-completed work. Concretely, on `2026-04-26T16:58:20Z` run `f5c0683a-25d8-4601-b52a-673a8e01aed9` invoked `vrooli scenario restart agent-manager` from a Bash tool; agent-manager died; the claude-code runner subprocess survived (it was placed in its own process group via `Setpgid: true` at `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go:132` and Node-based runtimes ignore SIGPIPE); claude-code finished the task at `17:00:41Z`, writing the full transcript to `~/.claude/projects/-home-matthalloran8--local-share-workspace-sandbox-20d4c57d-b31d-4876-9822-adab2f0da8c4-merged/1b673cb4-87c3-47da-b34a-6cbdf6981a61.jsonl`. The reconciler discovered the run had no heartbeat and at `17:15:15Z` (after `MaxRecoveryAge` of 10m) marked it `failed` with the message *"process terminated unexpectedly (detected by reconciler after 17m5s without heartbeat)"*. The 30 events emitted between agent-manager death and runner completion — three messages, thirteen tool_calls, fourteen tool_results, including the final summary — were lost from the run's event timeline despite existing in full on disk.

Three architectural facts cause this:

1. **The runner adapter consumes stdout via an in-process pipe.** When agent-manager dies, the read end goes with it. The runner may continue running but its output goes unread. (`scenarios/agent-manager/api/internal/adapters/runner/claude_code.go:221` `bufio.NewScanner(mp.Stdout())`; `codex_runner.go:382`; `opencode_runner.go:230`.)
2. **The reconciler's `AutoRecover` branch does not actually recover.** When it detects "process alive, executor heartbeat absent," it just declines to mark failed yet. There is no code that re-reads runner output. Eventually `MaxRecoveryAge` (10m) is hit and it kills the process and marks failed. (`reconciler.go:434-459`.)
3. **Only claude-code natively persists structured transcripts on disk.** Codex and opencode keep `SessionID` only in in-memory `map[uuid.UUID]string` fields (`codex_runner.go:100` `runThreadIDs`, `opencode_runner.go:42` `runSessionIDs`). When agent-manager dies, those maps are gone. Falling back to runner-native session files works for claude-code but not the other two.

## Scope (In / Out)

**In scope:**

- Per-run on-disk state directory under `scenarios/agent-manager/data/runs/<run-id>/` with `meta.json`, `transcript.ndjson`, `stderr.log`, `cursor.json`.
- Modify all three runner adapters' `Execute()` and `Continue()` paths so the runner's stdout is teed to `transcript.ndjson` (a tail consumer reads from the file rather than directly from the pipe). One shared helper `consumeTranscript(...)` per `parseFn` callback.
- Modify the reconciler's stale-run handling so that before marking a run failed it drains residual events from `transcript.ndjson` into the database via the same `parseFn`. Detect terminal events; mark complete or failed accordingly.
- New on-startup recovery pass: scan all runs in `running` status, load their state directories, drain residual events, decide each run's outcome (continue tailing, mark complete, mark failed).
- Backward-compat fallback for claude-code only: if `transcript.ndjson` is missing for a `running` run (e.g., the run started before this feature shipped), look up the runner-native transcript at `~/.claude/projects/<workspace-slug>/<session-id>.jsonl` and replay from there using the recovery script's translator (port the `/tmp/recover_run_f5c0683a.py` logic into Go).
- New CLI command `agent-manager run recover <run-id>` for manual one-shot recovery (uses the same code path as the reconciler's drain-and-finalize).
- Tests: unit (transcript writer, cursor advance, replay-from-cursor, parseFn-shared-with-live), integration (kill-and-restart agent-manager mid-run for each of the three runners), e2e (real `vrooli scenario restart agent-manager` from inside a run, verify recovery).
- Document the architecture in `scenarios/agent-manager/docs/runner-transcript-recovery.md` (new file).
- Garbage-collection: state directories for runs in any terminal state for >7 days are deleted by the reconciler.

**Out of scope:**

- Self-restart guards or CLI-level refusal of `vrooli scenario restart agent-manager` (see Hard Rule).
- Re-spawning the runner with `--resume <session-id>` if the runner is dead (we recover events, we do not re-run the model). This is a deliberate non-goal — re-running tools could double-apply changes.
- Live websocket re-broadcast of recovered events to UI clients that were watching at the time of the restart. Recovered events land in the database; existing subscription mechanics pick them up on next subscribe. Real-time replay during the restart window is a separate concern.
- A new runner type. No `RunnerType` registry changes.
- Changing `MaxRecoveryAge`, `StaleThreshold`, or other reconciler tunables. Those still gate "alive but quiet" runs; we just add a recovery path before they fire.
- Encrypting the on-disk transcript files. They sit in agent-manager's data dir alongside the SQLite DB, which is unencrypted today. Same threat model.
- Cross-host recovery (state directory on a different machine than the one that died). Single-host only.

## Current Technical Context

| File | Lines | Role |
|---|---|---|
| `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go` | 99-353 | claude-code `Execute()`. Spawns subprocess, scans `mp.Stdout()` line-by-line, calls `parseStreamEvents()`, emits to `req.EventSink`. |
| `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go` | 132 | `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` — runner is in its own process group; survives SIGTERM to agent-manager. |
| `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go` | 222 | `parseStreamEvents` is the runner-specific stream-line → `domain.RunEvent` translator. Same shape exists at `codex_runner.go:1316` (`parseCodexStreamEventsWithThreadID`) and `opencode_runner.go:748` (`parseStreamEventsWithSessionID`). These three are the per-runner integration points the new shared consumer will call into. |
| `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go` | 334 | Session ID captured only at end of `Execute()` from `streamStateFor(req.RunID).sessionID`. Needs to be persisted earlier — at first `system` event with `session_id` — so the recovery path can locate the runner-native fallback transcript. |
| `scenarios/agent-manager/api/internal/adapters/runner/codex_runner.go` | 1067-1092 | Codex `trackThreadID` / `threadIDForRun`. In-memory only. Same transition needed: persist on first emission. |
| `scenarios/agent-manager/api/internal/adapters/runner/opencode_runner.go` | 565-572 | OpenCode `trackSessionID`. Same. |
| `scenarios/agent-manager/api/internal/orchestration/reconciler.go` | 410-459 | `handleStaleRun`. Currently the only path for "process alive, executor gone." `AutoRecover` branch is a no-op. This is where the new replay path attaches. |
| `scenarios/agent-manager/api/internal/orchestration/reconciler.go` | 481-505 | `isProcessAlive` / `scanForProcess` — already detects living orphan runners by tag. Reusable. |
| `scenarios/agent-manager/api/internal/database/repository_run.go` | 685-738 | `eventRepository.Append()`. Same write path the recovery uses. |
| `scenarios/agent-manager/api/internal/database/schema.sql` | 60-100 | `runs` table — `session_id` column already exists. New columns to add: `runner_pid INTEGER`, `runner_pgid INTEGER`, `transcript_path TEXT`, `transcript_cursor INTEGER` (byte offset), `transcript_last_seq INTEGER`. |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go` | 1100-1165 | Existing comment block documents that agents may invoke `vrooli scenario restart <scenario>` — this remains the right behavior; the docstring should be updated to note the new recovery path. |
| `/tmp/recover_run_f5c0683a.py` | — | The one-shot recovery script for run `f5c0683a-…`. Contains the working translator from claude-code transcript JSONL → agent-manager event row. To be ported to Go for the backward-compat fallback. |

## Target End State

1. Every run started after this lands has a `<state-dir>/runs/<run-id>/transcript.ndjson` file written synchronously by the runner subprocess (via `cmd.Stdout` or a `tee`-style writer). The runner's stdout is durable on disk regardless of agent-manager's liveness.
2. Every run's `meta.json` is written atomically before the runner subprocess starts and contains `{run_id, runner_type, runner_pid, runner_pgid, working_dir, started_at, session_id?}`. `session_id` is appended (rewrite-by-rename) on first emission.
3. The `runs` table rows for in-flight runs carry `runner_pid`, `transcript_path`, and `transcript_cursor` so reconciliation can find the file without scanning the data dir.
4. agent-manager startup runs a recovery pass: for every run in `running` status, drain `transcript.ndjson` from `transcript_cursor` to EOF, emit each parsed event via `EventRepository.Append`, advance `transcript_cursor`, then decide:
   - If the runner pid is alive → continue tailing in a goroutine (the live-consumer path).
   - If pid is dead and the last event is a terminal `result` (or runner-equivalent) → mark complete with the runner-reported outcome.
   - If pid is dead and no terminal event present → mark failed with `"runner exited before terminal event"`.
5. The reconciler's `handleStaleRun` calls the same drain-and-finalize logic before its old "mark failed after MaxRecoveryAge" path.
6. The 30 events that were missed for run `f5c0683a-…` are present (they are, after the manual recovery; future incidents do not require manual intervention).
7. `vrooli scenario restart agent-manager` invoked from inside an agent-manager run: agent-manager exits, the runner subprocess survives, agent-manager comes back up within ~30s, the on-startup recovery pass discovers the in-flight run, attaches a tail consumer to `transcript.ndjson`, and continues streaming events as if nothing happened. The user observes a brief gap in event arrival but no run is marked failed and no events are lost.
8. `agent-manager run recover <run-id>` exists for operators: drains the transcript file once and prints the result. Idempotent — re-running on an already-recovered run is a no-op.
9. State directories for runs in terminal states older than 7 days are GC'd by the reconciler.
10. Architecture documented at `scenarios/agent-manager/docs/runner-transcript-recovery.md`.

## Implementation Strategy

Phased so each step is independently testable and revertable.

### Phase 1 — Schema + state directory layout

1. Add columns to `runs` via migration in `scenarios/agent-manager/api/internal/database/schema.sql` and a backfill script: `runner_pid INTEGER`, `runner_pgid INTEGER`, `transcript_path TEXT`, `transcript_cursor INTEGER DEFAULT 0`, `transcript_last_seq INTEGER DEFAULT 0`. `session_id` already exists.
2. Define the state dir convention: `<state-dir>/runs/<run-id>/{meta.json,transcript.ndjson,stderr.log,cursor.json}`. Resolve `<state-dir>` from existing config (same root as the SQLite DB).
3. Create a small package `scenarios/agent-manager/api/internal/runstate/` with `Open(runID) (*State, error)`, `WriteMeta(meta)`, `OpenTranscript() (*os.File, error)`, `Cursor()/AdvanceCursor()`, `Close()`. Atomic-write semantics for meta.json (write-and-rename).

### Phase 2 — Shared transcript consumer helper

4. Add `scenarios/agent-manager/api/internal/adapters/runner/transcript_consumer.go` exposing:

   ```go
   type ParseFn func(runID uuid.UUID, line string) []*domain.RunEvent
   type ConsumeArgs struct {
       RunID     uuid.UUID
       File      *os.File
       StartAt   int64                 // byte offset
       ParseFn   ParseFn
       Sink      EventSink
       Live      bool                  // true: tail; false: drain to EOF and return
       OnSessionID func(string)        // optional callback first time a session id is seen
       OnAdvance func(offset int64, lastSeq int64) // called after every successful emit so the caller can persist cursor
   }
   func Consume(ctx context.Context, args ConsumeArgs) error
   ```

   Internal implementation: read line-by-line; on each line, call `ParseFn`; emit via `Sink`; checkpoint cursor via `OnAdvance`. In live mode, on EOF wait for new bytes (poll or inotify) until `ctx` is done.

5. Wire each runner's existing `parseStreamEvents`-style function as a `ParseFn`. This is the seam — the per-runner translator code does not change.

### Phase 3 — Runner adapter conversion (one runner at a time, claude-code first)

6. In `claude_code.go:Execute()`:
   - Open `transcript.ndjson` for append.
   - Set `cmd.Stdout = transcript_file` (replaces the in-process pipe scanner). Stderr unchanged (still goes to `errorOutput.String()` for `ErrorMessage` reporting).
   - Spawn a goroutine running `Consume(ctx, ConsumeArgs{Live: true, ...})` against the same file, with the existing `r.parseStreamEvents` as `ParseFn`. Goroutine calls `OnAdvance` to update `transcript_cursor` in the run row.
   - Wait for the subprocess via `mp.Wait()` as today, then drain remaining lines from the file (`Live: false`), then return `ExecuteResult` exactly as today.
   - First call to `OnSessionID(id)`: persist `session_id` to the runs row immediately (existing column). Allows recovery to locate the runner-native fallback transcript.
   - Persist `runner_pid` and `runner_pgid` immediately after `cmd.Start()` succeeds.

7. Repeat for `codex_runner.go:executeWithJSONStream()` (line 286) and `opencode_runner.go:Execute()` (line 127). Same shape, different `ParseFn`. The `executeWithWrapper` codex path is non-streaming text — for that path emit one synthesized log event per line into transcript.ndjson with a fixed `{"type":"line","text":"..."}` envelope and write a corresponding parser.

8. Also convert each runner's `Continue()` path identically. Continuation runs reuse the existing transcript file (append) so resume-after-restart of a continuation works the same way.

### Phase 4 — On-startup and reconciler recovery

9. Add `scenarios/agent-manager/api/internal/orchestration/recovery.go` with `RecoverInFlightRuns(ctx) error`. Iterates runs with `status=running`; for each:
   - Read state directory; if missing, log + skip (the run failed to ever start writing — that's a different bug).
   - Open `transcript.ndjson`, seek to `transcript_cursor`, drain remaining lines into the events table using the runner's `ParseFn`.
   - `kill -0 runner_pid` (treat ESRCH as dead, EPERM as alive).
   - Alive: spawn a live tail goroutine via the same `Consume` helper. Run continues.
   - Dead: examine drained events for a terminal marker (claude-code: a `result` event; codex: a `turn.completed` followed by clean EOF; opencode: a `step_finish` with terminal `Reason`). If found, mark `status=complete`, populate `Summary`, set `ended_at`. If not, mark `status=failed` with `"runner exited before terminal event"`.

10. Call `RecoverInFlightRuns(ctx)` from `service.go` startup (after DB open, before HTTP listener starts).

11. In `reconciler.go:handleStaleRun`, before the `MaxRecoveryAge` check at line 443: call the same drain step and re-evaluate. If draining produced terminal events, mark complete and return without killing the runner.

### Phase 5 — Backward-compat fallback (claude-code only)

12. When recovery finds `transcript.ndjson` missing but the run has `runner_type=claude-code` and `session_id` set, locate the runner-native file at `~/.claude/projects/<workspace-slug>/<session-id>.jsonl`. Workspace-slug is derived from `working_dir` by claude-code's deterministic rule (replace `/` with `-`, prepend `-home-...`).
13. Port the translator from `/tmp/recover_run_f5c0683a.py` into Go as `recovery/claude_native.go`. Same translation logic — `assistant.tool_use` → tool_call event; `user.tool_result` → tool_result event; `assistant.text` → message event. Drain via `EventRepository.Append`.
14. No equivalent fallback exists for codex/opencode (they don't write structured per-session files on disk). For runs of those types started before this feature, the recovery path simply marks them failed — a one-time cost.

### Phase 6 — Operator CLI and GC

15. Add `agent-manager run recover <run-id>` subcommand. Calls the same recovery code path as the on-startup pass, scoped to one run. Idempotent.
16. Add a reconciler tick (or new sweeper) that deletes state directories for runs in any terminal state that have not been updated in >7 days.

### Phase 7 — Documentation

17. Write `scenarios/agent-manager/docs/runner-transcript-recovery.md` covering: state directory layout, the lifecycle of an in-flight run across an agent-manager restart, how to recover manually, where the runner-native fallback applies.
18. Update the docstring at `run_executor.go:1100-1165` to note that agents *may* call `vrooli scenario restart agent-manager` and that recovery is automatic.

## Contract Decisions

| ID | Topic | Resolution |
|---|---|---|
| D1 | Should agent-manager re-spawn dead runners with `--resume <session-id>` to continue work? | **No.** Re-running tools could double-apply changes. Recovery is event-replay only; if the runner died mid-task, the run is marked failed. |
| D2 | Where do state directories live? | `<state-dir>/runs/<run-id>/`, where `<state-dir>` is the parent of the SQLite DB path. Single-host only — no cross-host state sync. |
| D3 | Transcript file format. | Verbatim runner stdout (one JSON object per line for streaming runners; one text-line per line for `executeWithWrapper`). The runner's own `parseStreamEvents` works against it unchanged. We do *not* re-encode into a unified format — that would create a second translation layer. |
| D4 | Cursor representation. | Byte offset into `transcript.ndjson`. Plus `transcript_last_seq` (the last DB-side `run_events.sequence` we wrote) so we can detect partial-write situations after a crash mid-batch. |
| D5 | Should the live consumer use `inotify` or polling? | Polling at 100ms intervals. Lower complexity, cross-platform, perfectly adequate for human-paced agent output. Revisit only if profiling shows it matters. |
| D6 | What about runs started before this feature? | Migration leaves `transcript_path` NULL on existing rows. Recovery paths skip such rows (`status=running` rows pre-feature are reconciled the old way). For claude-code specifically, the runner-native fallback can salvage if invoked manually via `agent-manager run recover`. |
| D7 | Backward compatibility shim. | None. This is a greenfield change to the runner adapter internals. Existing `Runner` interface (`interface.go:36-72`) is unchanged. |
| D8 | Self-restart guard. | **Forbidden.** See Hard Rule. |

## Testing Plan

Tests live next to the code they cover. Run via `cd scenarios/agent-manager && make test` and `vrooli scenario test agent-manager`.

| # | Behavior | File | Layer |
|---|---|---|---|
| 1 | `runstate.Open` creates dir, atomic-writes meta.json, opens transcript for append | `internal/runstate/runstate_test.go` | Unit |
| 2 | `Consume` in non-live mode parses every line, emits to sink, advances cursor | `internal/adapters/runner/transcript_consumer_test.go` | Unit |
| 3 | `Consume` in live mode tails the file and reacts to new bytes within poll interval | same | Unit (with goroutine + `time.Sleep` driver) |
| 4 | `Consume` resumes mid-file from a non-zero `StartAt`, skipping prior bytes | same | Unit |
| 5 | claude-code `parseStreamEvents` produces the same events when called with file-derived lines as with pipe-derived lines (regression — confirms the seam) | `internal/adapters/runner/claude_code_consumer_test.go` | Unit |
| 6 | Same for codex `parseCodexStreamEventsWithThreadID` | `internal/adapters/runner/codex_consumer_test.go` | Unit |
| 7 | Same for opencode `parseStreamEventsWithSessionID` | `internal/adapters/runner/opencode_consumer_test.go` | Unit |
| 8 | claude-code Execute end-to-end with a fake `claude` binary writes events to file AND emits via sink AND advances cursor | `internal/adapters/runner/claude_code_test.go` | Integration |
| 9 | Same for codex (with fake `codex` binary) | `internal/adapters/runner/codex_runner_test.go` | Integration |
| 10 | Same for opencode (with fake `opencode` binary) | `internal/adapters/runner/opencode_runner_test.go` | Integration |
| 11 | `RecoverInFlightRuns` on a run whose runner pid is alive: drains residual events, attaches live tail | `internal/orchestration/recovery_test.go` | Integration |
| 12 | `RecoverInFlightRuns` on a run whose runner pid is dead with terminal event in transcript: marks complete | same | Integration |
| 13 | `RecoverInFlightRuns` on a run whose runner pid is dead with no terminal event: marks failed | same | Integration |
| 14 | Reconciler `handleStaleRun` calls drain step before `MaxRecoveryAge` kill | `internal/orchestration/reconciler_test.go` | Integration |
| 15 | claude-code backward-compat fallback: runs with no `transcript.ndjson` but with `session_id` recover from `~/.claude/projects/<slug>/<session-id>.jsonl` | `internal/orchestration/recovery_claude_native_test.go` | Integration |
| 16 | E2E for each runner: start a real run, kill agent-manager via SIGTERM mid-execution, restart agent-manager, verify the run reaches `complete` status with all events present | `internal/orchestration/recovery_e2e_test.go` (build-tag-gated, runs against real CLIs) | E2E |
| 17 | E2E specifically reproducing the original incident: a claude-code run invokes `vrooli scenario restart agent-manager` from a Bash tool; verify recovery is automatic | same file | E2E |
| 18 | `agent-manager run recover <run-id>` is idempotent: running twice is safe | `cmd/agent-manager/run_recover_test.go` | Integration |
| 19 | GC: state directories for runs in terminal status older than 7 days are deleted | `internal/orchestration/gc_test.go` | Integration |
| 20 | No self-restart guard exists: `vrooli scenario restart agent-manager` invoked from a sandbox shell while a run is in progress is not refused | `internal/orchestration/recovery_e2e_test.go` (assertion-only) | E2E |

The kill-and-restart scaffolding for tests #16-#17 lives in `recovery_e2e_test.go`. Pattern: use `os/exec` to spawn agent-manager as a child of the test, send SIGTERM to its pgid, wait for exit, restart, poll the run via the API until it reaches a terminal status, assert event count and content.

## Rollout / Validation Checklist

1. **Phased per-runner rollout**:
   - [ ] Phase 1-2 (schema + shared helper) merge first, no behavior change.
   - [ ] Phase 3 lands one runner at a time. After each, run the corresponding E2E test.
   - [ ] Phase 4-5 (recovery on startup + claude-code fallback) merges as one PR.
2. **Pre-merge gates** (each PR):
   - [ ] `cd scenarios/agent-manager && make test` passes locally and in CI.
   - [ ] `golangci-lint run ./scenarios/agent-manager/...` clean.
   - [ ] `gofumpt -l scenarios/agent-manager` reports no diffs.
   - [ ] Documentation updated alongside code.
3. **Post-deploy validation** (single deploy of the staged stack):
   - [ ] Start a long-running claude-code task.
   - [ ] Mid-execution, run `vrooli scenario restart agent-manager` (the operation that originally caused this incident).
   - [ ] Observe in `vrooli scenario logs agent-manager` that the on-startup recovery pass logs `[recovery] resumed run <id> from transcript at offset <N>`.
   - [ ] Wait for the task to complete; confirm via `agent-manager run get <id>` that `Status: complete` and `Summary.Description` contains the agent's final message.
   - [ ] Repeat for a codex run and an opencode run.
4. **Backfill validation**:
   - [ ] Run `agent-manager run recover f5c0683a-25d8-4601-b52a-673a8e01aed9` — should report `idempotent: no events to recover` (already manually recovered).
   - [ ] Pick another historically-failed claude-code run (any from `git log` referencing this issue) and run `agent-manager run recover` — verify the runner-native fallback path successfully recovers it.
5. **Post-rollout cleanup**:
   - [ ] Delete `/tmp/recover_run_f5c0683a.py` from local scratch.

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Runners write enormous lines (e.g., a tool result with megabytes of output). The existing 10MB scanner buffer at `claude_code.go:220` already accommodates this — the new file-based consumer must use the same limit. | Medium | Medium | Use an identically-configured `bufio.Scanner` in `Consume`. Add Test #4 case covering an 8MB line. |
| `cmd.Stdout = file` changes the runner's stdout from a pipe to a file. Some runners might detect "stdout is a tty" and behave differently. | Low | Low | All three runners are run with `--print`/`--json` modes that already disable any tty-aware formatting. Confirm via Test #8-10 that event output is byte-identical to the pipe-based path. |
| Polling-based tail (D5) could miss the very-last-line of a runner that exits within one poll interval. | Low | Medium | After `mp.Wait()` returns, do one final non-live `Consume` drain to EOF before declaring the run done. Already in Strategy step 6. |
| Agent-manager dies between `cmd.Start()` and writing `meta.json` — runner orphaned with no record. | Low | Low | Atomic-write `meta.json` *before* `cmd.Start()`, rolling back the file on Start failure. Worst case: agent-manager finds a meta.json with no live process; recovery treats this as "runner failed to ever start writing" and marks failed. |
| Disk fills up during a long-running run — runner gets EIO writing to transcript, dies. | Low | Medium | Same outcome as any other unexpected runner death — recovery sees no terminal event and marks failed. Add a metric `transcript_write_errors_total` so this is visible. Disk-full is an operator concern, not specific to this design. |
| The runner-native fallback's translator (claude_native.go) drifts from claude-code's transcript format if Anthropic changes the schema. | Medium | Low | Fallback is a backward-compat affordance; primary path is the agent-manager-managed transcript. If the fallback breaks, primary recovery still works for new runs. Pin a regression test using the current transcript format (Test #15). |
| Polling tail fights with the SQLite writer for fsync bandwidth on slow disks. | Low | Low | Cursor advancement is one row update per N events (batch every 50 lines or 500ms). Tunable. |
| Race: agent-manager restarts twice quickly; on the second startup a tail goroutine started by the first startup is still alive. | Low | Low | The tail goroutine runs in-process; if agent-manager exits, its goroutines exit too. The new agent-manager process starts a fresh tail. No race possible across processes. |
| The new recovery path masks a real bug — a runner that genuinely exits unexpectedly mid-task gets its events recovered and looks "complete" when it should be "failed." | Medium | Medium | Recovery only marks a run `complete` if a terminal event is present in the transcript. No terminal event → `failed`. Add Test #13 to pin this. |

## Non-goals / Prohibited Patterns

- **No self-restart guard.** Restated: `vrooli scenario restart agent-manager` from inside a run must remain allowed.
- **No re-spawning the runner with `--resume` on death.** Recovery is event-replay only.
- **No new runner type or change to the `Runner` interface.** Adapter internals only.
- **No cross-host state.** Single-host design.
- **No encryption of transcript files.** Same threat model as the SQLite DB.
- **No backwards-compatibility flag.** State-dir writing is on for all new runs unconditionally.
- **No live websocket replay during the restart window.** Out of scope; events land in DB, UI re-subscribes.
- **No retention beyond 7 days for terminal-state state directories.** GC is mandatory.
- **No sweeping of runs that were `running` before the feature shipped via the new pipeline.** Those are pre-existing failures and are handled (or not) by the manual `agent-manager run recover` path.
- **No introduction of a second event format.** Transcript is verbatim runner stdout; the runner's own `parseStreamEvents` is the only translator.

## Definition of Done

- [ ] Schema migration adds `runner_pid`, `runner_pgid`, `transcript_path`, `transcript_cursor`, `transcript_last_seq` to `runs`.
- [ ] `runstate` package exists with atomic meta.json write and append-mode transcript open.
- [ ] `transcript_consumer.go` provides a single `Consume` helper used by all three runners' live path AND by `recovery.go`.
- [ ] All three runners' `Execute()` and `Continue()` write to `transcript.ndjson` and consume from it via `Consume`.
- [ ] `RecoverInFlightRuns` runs at agent-manager startup and reattaches or finalizes every `running` run.
- [ ] `reconciler.go:handleStaleRun` drains the transcript before its `MaxRecoveryAge` kill path.
- [ ] `agent-manager run recover <run-id>` exists and is idempotent.
- [ ] claude-code backward-compat fallback (runner-native transcript translator) ports the working logic from `/tmp/recover_run_f5c0683a.py` into Go.
- [ ] All 20 tests in the Testing Plan pass.
- [ ] E2E test #17 (the exact original incident: claude-code calls `vrooli scenario restart agent-manager` from a Bash tool, run completes successfully with all events) passes.
- [ ] **No self-restart guard has been added.** This is asserted as a test (#20).
- [ ] `scenarios/agent-manager/docs/runner-transcript-recovery.md` exists and explains the architecture.
- [ ] `vrooli scenario restart agent-manager` invoked from inside a real run during post-deploy validation produces no failed runs and no lost events.
- [ ] `golangci-lint run` and `gofumpt -l` clean across all changes.
- [ ] State-directory GC sweep deletes terminal-state dirs older than 7 days.
