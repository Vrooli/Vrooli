# Failed Run and Workflow Triage

Use this procedure to diagnose completed failures from durable evidence only.
Do not restart, retry, or re-run work while collecting evidence: doing so can
overwrite the ordering and timing facts needed to identify the original cause.

## Evidence order

1. Read the run summary and ordered run event log: `GET /api/v1/runs/{id}` and
   `GET /api/v1/runs/{id}/events`.
2. Read the persisted transcript audit: `GET /api/v1/runs/{id}/audit-transcript`.
3. For workflow work, read the execution trace: `GET /api/v1/workflow-executions/{id}/trace`.
   The trace is the workflow journal: wait registrations, signals, attempts,
   structured-result validation, and child-workflow outcomes are ordered there.
4. Correlate by run id, workflow execution id, attempt id, event sequence, and
   timestamps. Prefer typed event payloads over process logs.

## Testing ladder

Diagnose a defect with the cheapest evidence that can answer it, in this order:

1. Contract and conformance tests (no process launch).
2. Codec replay against recorded transcripts.
3. Fake-agent process replay through the real launcher and persistence path.
4. Local-model smoke using the configured local provider.
5. A real agent run.

Do not spend agent tokens to diagnose behavior that a query, recorded replay,
or fake-agent process test can prove. Escalate to a real run only after the
lower rung cannot reproduce or distinguish the issue.

## Run diagnosis

| Evidence | Meaning | Next action |
| --- | --- | --- |
| Terminal error event with a panic stack | A protected execution seam recovered a panic and persisted it as a failed run. | Classify the stack's first application frame; fix the owning phase, not the recovery wrapper. |
| `pending` reap event | Dispatch never progressed before the configured liveness threshold. | Check preceding create/dispatch events and dispatcher capacity; do not assume the runner executed. |
| `heartbeat.miss` then stale/recovery events | The run stopped making durable progress. | Compare last transcript entry, last heartbeat, and sandbox/runner events to distinguish process loss from database failure. |
| `checkpoint.failure` or append-failure diagnostic | State/evidence persistence failed. | Treat the event sequence as incomplete; repair storage availability before retrying. |
| Fallback or policy-candidate events | A runner/model selection decision occurred. | Use the ordered candidate payloads and catalog digest, rather than transcript position, to identify the chosen runtime. |

### Worked example: injected runner panic

1. The run summary is `failed`.
2. The final run event is an error whose details include a stack trace and the
   recovered phase/dispatch context.
3. The transcript has no later terminal agent output.

Conclusion: the injected runner/phase panic caused the failure; recovery
worked because the process survived and the terminal event is durable. The
fix belongs to the stack's first application frame.

## Workflow diagnosis

1. Start from the workflow execution trace, not from a child transcript.
2. Locate the last journal entry for the failed node attempt.
3. Read that attempt's child run events/transcript when an attempt references a
   run id.
4. Compare journal sequence with signal/wait entries before declaring a wait
   lost. A wait is armed only when its journal entry says so.

### Worked example: failing result specification

1. The trace shows a node attempt with raw output followed by a structured
   result validation error.
2. The child transcript contains the same raw output; no later successful
   attempt exists.
3. The execution terminal result records the node failure.

Conclusion: the result specification rejected the produced value, rather than
the runner or scheduler failing. Correct the producer output or declared
schema, then create a new attempt through the normal workflow operation.

## Escalation record

When escalating, include the immutable identifiers and sequence range, the
last terminal/diagnostic event payload, the relevant journal entries, and the
transcript excerpt reference. Do not attach a new rerun as evidence for the
original incident.

## Forced-failure drill

Use the injected failures below to verify the evidence path without modifying
production data or starting a real agent run:

```bash
cd scenarios/agent-manager/api
go test ./internal/orchestration -run 'TestRecoverPanickedRunFailsOnlyAffectedRunWithStackEvent|TestAppendAndBroadcastEvents_DoesNotBroadcastWhenAppendFails|TestBroadcastingEventSink_DoesNotBroadcastWhenAppendFails|TestWorkflowExecutionControlAndTriageProjections' -count=1 -v
```

The panic case must leave the affected run terminally failed and persist its
stack-bearing event. The append-failure cases must return the storage error and
must not broadcast an event that was not durably appended. The workflow case
injects a terminal workflow failure, verifies its trace and control projection,
then verifies the idempotent retry path. These assertions are the offline
counterpart of the diagnosis rows above; inspect their named test output rather
than re-running an incident.

See also [EVENT_TAXONOMY.md](EVENT_TAXONOMY.md), [SEAMS.md](SEAMS.md), and
[TEMPORAL-FLOWS.md](TEMPORAL-FLOWS.md).

## Investigation-quality baseline inventory — 2026-07-28

Baseline: Test Genie run `20260728-211850-c48832d6` (FAIL; inherited) and
`storage-health validate scenario agent-manager` (0 findings).

The baseline suite has inherited failures in `structure`, `contracts`, `api`,
`dependencies`, `quality`, `docs`, `unit`, `security`, `measures`, and `proto`.
They are not evidence of a regression in this plan. The investigation-path
inventory below is the work queue for this execution.

| Entry | Classification | Evidence / disposition |
| --- | --- | --- |
| Duplicate investigation snapshot omits terminal result provenance and structured-result diagnostics | blocking | `orchestration/investigation.go` independently renders a partial overview; replace it with the shared report projection. |
| Typed operational events are rendered as clipped JSON rather than discriminators | blocking | `formatEventSummary` has no cases for fallback, health, sandbox, heartbeat, checkpoint, or retry events. |
| Raw unified diffs can consume the investigation context budget | blocking | The current diff attachment contains `UnifiedDiff` and the rendered attachment set is byte-truncated. |
| Investigation contract skill restates workflow schema and lacks outcome selection rules | blocking | The workflow-node skill is currently a tools-mode method prompt rather than a contract. |
| Investigation profile cannot invoke the project CLI | blocking | Its tool allowlist lacks `shell`, so documented CLI evidence commands are unreachable. |
| A run can call lifecycle operations that create or stop other runs | blocking | Shell is necessarily broad; enforce the run identity boundary at HTTP handlers. |
| Per-run SQL statistics cannot be scoped to a specific run | blocking | `repository.StatsFilter` has no run-id predicate. |
| Typed-event aggregate is process-global only | blocking | `stats.Engine` has no throwaway per-run fold. |
| Observed receipts are handler-only and unavailable to orchestration | blocking | The receipts dependency is held by `handlers.Handler`, preventing a shared report reader. |
| Recurrence and successful-run efficiency evidence are not queryable | deferred | Implement only after the single-run report and live validation are sound; do not let cross-run work delay correctness. |

## Investigation-quality live validation — 2026-07-29

Two independent investigation runs against the live identity-guard probe,
`b94c5723-eaea-44eb-98c9-67adbf659662` and
`cfed57b6-ffda-4e90-9fa4-10175ceae806`, selected `Both` with `High`
confidence. Both identified the same bounded correction: retain the API guard,
refuse operator-only lifecycle commands in the CLI before HTTP dispatch, and
make the investigation contract direct agents not to retry a refused operation.

The corrections are covered by handler and CLI tests. Compact redacted replay
fixtures preserve each live run in
`api/internal/adapters/runner/codecs/testdata/corpus/` as
`codex-live-investigation-both-{1,2}.jsonl`; the full source transcripts were
harvested with `api/cmd/harvest-replay` before fixture compaction.

The remaining broad gate is not an investigation defect: the scenario's UI
coverage policy is 85%, while the full UI suite currently measures about 62%.
The policy has not been weakened; this remains tracked as `P-007` and requires
additional behavior tests across the existing UI surface.
