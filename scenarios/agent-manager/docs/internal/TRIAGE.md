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
