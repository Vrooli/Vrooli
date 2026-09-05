# Runner Transcript Recovery

`agent-manager` now keeps a per-run durable transcript so a runner can outlive an `agent-manager` restart without losing already-emitted work.

## State layout

Each in-flight run gets a state directory under:

`<agent-manager data dir>/runs/<run-id>/`

Files:

- `meta.json`: run id, runner type, pid/pgid, working directory, started time, session id
- `transcript.ndjson`: verbatim runner stdout
- `stderr.log`: stderr side-channel for diagnostics
- `cursor.json`: last drained byte offset and last persisted event sequence

The SQLite `runs` row mirrors the critical recovery fields:

- `runner_pid`
- `runner_pgid`
- `transcript_path`
- `transcript_cursor`
- `transcript_last_seq`
- existing `session_id`

## Execution flow

When a run starts, `agent-manager` allocates the state directory before the runner process is started. Runner stdout is redirected to `transcript.ndjson`, and a transcript consumer tails that file and feeds the normal event store / websocket broadcast path. As lines are consumed, the consumer advances the byte cursor and persists the first discovered runner session id immediately.

This means the durable record belongs to `agent-manager`, not to an in-memory stdout pipe.

## Restart recovery

On startup, the reconciler runs `RecoverInFlightRuns` before the normal loop starts.

For each `running` run it:

1. Reopens the transcript from `transcript_cursor`
2. Replays any missing events into the database through the same runner parser used during live execution
3. Checks whether `runner_pid` is still alive
4. If alive, starts a polling tailer so new transcript bytes continue flowing into the event stream
5. If dead, finalizes from the terminal transcript signal when one exists, otherwise marks the run failed with `runner exited before terminal event`

The stale-run reconciler path uses the same drain step before deciding to kill or fail a run.

## Interactive runs

Interactive runs (`ExecutionMode == interactive`, see
[interactive-runner-design.md](interactive-runner-design.md)) reuse the same
durable-transcript machinery — a cursor over the agent-owned on-disk transcript,
drained through the codec transcript parser — but their **liveness signal is the
web-console session, not a local pid**. The CLI runs inside a web-console tmux
session, so there is no `runner_pid` for the reconciler to scan.

On startup `RecoverInFlightRuns` / `handleStaleRun` route interactive runs to
`recoverInteractiveRun`, which:

1. drains the transcript from the persisted cursor (the identical drain step);
2. if a **failure terminal** was already written, finalizes the run Failed;
3. otherwise calls `SessionsService.GetSession` on the stored
   `web_console_session_id`:
   - **gone** → finalize (Complete if a success terminal was already seen, else
     Failed with `web-console session <id> no longer exists; interactive run
     cannot be recovered`);
   - **alive** → reattach the tailer from the cursor (no duplicate events) and
     let its turn-boundary idle-debounce drive true completion.

`handleStaleRun` returns early for interactive runs, so the pgid / MaxRecoveryAge
kill path never fires against a session-hosted CLI. A session that vanishes
mid-tail is caught by a background `GetSession` watcher that cancels the tail.
When the reconciler has no web-console client wired, interactive recovery is a
logged idempotent no-op — it never falsely completes or fails a run. Known
limitation: a codex rollout that **rotated** across the restart is not re-followed
(the run dir is not persisted); the pinned rollout path still tails correctly
within a session.

## Manual recovery

Operators can run:

```bash
agent-manager run recover <run-id>
```

That uses the same drain-and-finalize path as startup recovery and is idempotent.

## Imported session corpus

`agent-manager run import-session-corpus` adopts a bounded research corpus
from the runner session stores declared by the relevant resource manifests. It
does not expose a host path to callers and defaults to Codex and Claude Code.

```bash
agent-manager run import-session-corpus --per-month 1 --limit 24 --json
agent-manager run replay-invocation-corpus --tag-prefix agent-manager-imported --limit 100 --json
agent-manager measures select-cohort --window this_week --json
agent-manager run episode-cohort --tag-prefix agent-manager-imported --limit 100 --json
```

Selection is reproducible: within each runner/month it takes pathname-sorted
sessions, then round-robins runners with each runner's months ascending. Every
imported row records `import_source_harness`, `import_source_session_id`, and
`imported_at`; the first two form a unique identity, so repeating the command
reports `alreadyImported` instead of duplicating evidence. The response reports
selected, imported, already-imported, replayed, unreplayable, failed, and every
skip reason. A metadata-only transcript has no retained event timestamp and is
explicitly reported as `unreplayable`, never silently treated as a projection.

## Claude native fallback

For older Claude runs created before `transcript.ndjson` existed, manual recovery can fall back to the runner-native transcript under `~/.claude/projects/.../<session-id>.jsonl` when the run still has a stored `session_id`.

There is no equivalent historical fallback for Codex or OpenCode.

## Retention

The reconciler deletes run state directories for terminal runs older than 7 days.
