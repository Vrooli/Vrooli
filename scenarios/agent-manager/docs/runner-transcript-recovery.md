# Runner Transcript Recovery

`agent-manager` now keeps a per-run durable transcript so a runner can outlive an `agent-manager` restart without losing already-emitted work.

## State layout

Each in-flight run gets a state directory under:

`<agent-manager data dir>/runs/<run-id>/`

Files:

- `meta.json`: run id, runner type, pid/pgid, runner identity, owner epoch, working directory, started time, session id
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

The state root is a startup prerequisite. It must be an absolute writable path
outside process-ephemeral locations such as `/tmp`, `/var/tmp`, `/run`, and
`/dev/shm`; startup performs a synced write probe before admitting work. The
root is durable across an Agent Manager process restart. Host reboot or storage
loss is outside this guarantee and requires provider/session evidence or a
fresh qualified recovery.

Each launch binds the state to the run lifecycle epoch and a stable runner
identity. Recovery opens existing metadata rather than replacing it, so the
identity, owner epoch, process binding, session, and cursor remain evidence for
the replacement owner. Lifecycle-version compare-and-swap writes prevent a
stale owner from advancing or finalizing a run after reattachment.

This means the durable record belongs to `agent-manager`, not to an in-memory stdout pipe.

## Restart recovery

On startup, the reconciler runs `RecoverInFlightRuns` before the normal loop starts.

For each `running` run it:

1. Reopens the transcript from `transcript_cursor`
2. Replays any missing events into the database through the same runner parser used during live execution
3. Checks whether `runner_pid` is still alive
4. If alive, starts a polling tailer so new transcript bytes continue flowing into the event stream
5. If dead, finalizes from the terminal transcript signal when one exists; otherwise preserves the run as `needs_review` with typed `interruption/crash` state and the actionable handoff `runner exited before terminal event`

The stale-run reconciler path uses the same drain step before deciding what to
report. A positively identified detached runner is never killed merely because
its heartbeat is older than `MaxRecoveryAge`; that threshold is diagnostic so a
control-plane restart or long provider call cannot become data loss. Explicit
cancellation and destructive maintenance retain termination authority.

## Recovery-first restart procedure

An ordinary Agent Manager restart is safe while detached runners are active:

1. Confirm the run is `running`, `parked`, or `needs_review`, and note its run
   ID from the run report.
2. Run `vrooli scenario restart agent-manager --force-lifecycle
   --lifecycle-override-reason "recovery-first restart; retain identified
   detached runners"`. The audited recovery-first path restarts the control
   plane and reconciler without authorizing termination of a positively
   identified detached runner.
3. Wait for Agent Manager health and startup recovery to report ready.
4. Inspect the run report and recovery events. A reattached run should retain
   its runner identity, owner epoch, transcript cursor, checkpoint, and prior
   progress.
5. If the runner disappeared without terminal evidence, resolve the resulting
   `needs_review` interruption explicitly. Do not start a duplicate turn until
   effect and terminal evidence review is complete.

If the owner is still starting and returns `503 startup recovery is not ready`,
the same explicitly audited command may adopt the already-closed maintenance
fence and continue the replacement. A generic unavailable-owner response, an
identified current executor, or an open admission fence still refuses the
restart. This distinction is important: startup-recovery uncertainty is handed
to the new owner for reattachment, while destructive operations continue to
require complete inventory evidence.

This procedure does not promise survival of a host reboot or loss of the
durable state provider. Destructive maintenance and explicit run cancellation
remain separate operations and may terminate the runner after their authority
and evidence checks succeed.

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
   - **gone** → finalize (Complete if a success terminal was already seen,
     otherwise preserve `needs_review` with typed `interruption/crash` state);
   - **alive** → reattach the tailer from the cursor (no duplicate events) and
     let its turn-boundary idle-debounce drive true completion.

`handleStaleRun` returns early for interactive runs, so the pgid / MaxRecoveryAge
kill path never fires against a session-hosted CLI. A session that vanishes
mid-tail is caught by a background `GetSession` watcher that cancels the tail.
When the reconciler has no web-console client wired, interactive recovery is a
logged idempotent no-op — it never falsely completes or fails a run. Interactive
transcript bindings are retained in the durable run directory, so a rotated
Codex rollout is followed from the persisted session binding when the provider
exposes the replacement path. If the provider does not expose that path, the
run remains explicitly recoverable rather than silently restarting the turn.

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

## Continuity evidence

The disposable RCL continuity canary used two active workflow executions while
Agent Manager was restarted through the lifecycle command. The positive RCL
workflow was `2218b0f2-eb7d-4750-8456-24ec2fecd9b6` (Agent Manager execution
`21e1a82c-136b-4b8c-9192-bab0b440a16c`) and the unrelated workflow was
`9bf4ffba-323b-4faf-8c52-ebe0dd3e67ab` (execution
`4fb6582d-e6b9-4457-ba2f-c9489a626172`). Both executions were rehydrated under
new run records without unknown inventory; the positive workflow was refreshed
through `RefreshWorkflow` and reported `WORKFLOW_STATUS_SUCCEEDED` with a null
error. The recovery run itself may report `needs_review` when the disposable
fixture cannot execute its local build obligations; that is an explicit
execution limitation, not permission to claim provider-side work completed.

This evidence proves restart, ownership rehydration, and durable workflow
projection for the fixture. It does not prove survival across host reboot,
provider-side rollback, or a missing durable database. Those cases remain
subject to the limitations above and require explicit reconciliation.

A subsequent tracked-mode rerun (`be99514f-d770-4c15-a9f4-2e2da0ca624c`, Agent
Manager execution `4e87e28d-a223-4db3-9a45-a77a1044338d`, runner
`5abd56a7-55dc-450d-8c6f-a35bf1673a16`) reached terminal accounting with
`accounting_complete=true`, but its sandbox inspection reported `Files: 0` and
only `.vrooli-shadowed-view-do-not-use` in the merged workspace. The assistant
therefore reported blocked validation because the physical fixture root and
shared `packages/` dependency tree were unavailable. This is recorded as
scenario-qa bug `knw-1789463027738566031`; it is provider evidence, not a
successful fixture build receipt. Until that bug is resolved, RCL canary
readiness remains conditional on the workspace-sandbox provider.

The disposable fixture now has lifecycle-owned API and UI processes, complete
health responses including the runtime build identity, and starts successfully
through `vrooli scenario setup/start --path`. Test Genie subsequently collected
receipt `20260915-093346-bb1b7947` with all 26 requested phases executed (11
passed, 14 failed, 1 skipped). The receipt is useful boundary evidence, but it
is not a green readiness receipt: the failures are fixture contract debt (for
example missing CLI manifest and template-era UI/proto obligations), not proof
that Agent Manager continuity failed. The sandbox canary still cannot use this
host receipt as proof that edits inside a path-illusion workspace were tested;
that requires a sandbox-aware Test Genie execution boundary.

RCL now also projects the assistant's typed `outcome` into its durable status:
`completed` is required for `succeeded`; `blocked`, `needs_review`, and
`failed` become failed projections with the typed outcome in `error`. Earlier
canary records that show `WORKFLOW_STATUS_SUCCEEDED` with a blocked summary
were created before this projection guard and must not be used as success
receipts.

After the fixture gained lifecycle-owned health surfaces, a new canary
(`71974206-ec22-4470-84b3-668e8a2ef3cb`, Agent Manager execution
`218ff30c-656f-471d-911e-8e50fb9613dd`) returned `outcome=completed`, with
RCL obligations and before/after checks passing. Its `manualReview=true`
policy intentionally skipped sandbox finalization and exposed a separate
read-after-disconnect gap: the RCL row stayed active until the service was
restarted and its recovery-aware `Get` reconciled the terminal Agent Manager
execution. The row then persisted `WORKFLOW_STATUS_SUCCEEDED`. This proves
the durable projection can converge after a disconnected waiter; it does not
turn the red generic Test Genie fixture receipt into a release-readiness pass.
