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

## Manual recovery

Operators can run:

```bash
agent-manager run recover <run-id>
```

That uses the same drain-and-finalize path as startup recovery and is idempotent.

## Claude native fallback

For older Claude runs created before `transcript.ndjson` existed, manual recovery can fall back to the runner-native transcript under `~/.claude/projects/.../<session-id>.jsonl` when the run still has a stored `session_id`.

There is no equivalent historical fallback for Codex or OpenCode.

## Retention

The reconciler deletes run state directories for terminal runs older than 7 days.
