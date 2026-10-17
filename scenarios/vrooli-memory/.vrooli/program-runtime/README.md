# Shared learning programs

These workflows compose existing Vrooli Memory APIs. Source Ledger remains the
storage authority. The programs do not provision scopes, invoke a model, infer
task success, or own a checkpoint/outbox. Program Runtime owns execution checkpoints
and frozen capture delivery; `scope-bootstrap` is independent.

## Automatic task boundary

Call a scenario program with `learning_task` metadata normally. Runtime prepares
bounded scoped advice, checkpoints before domain execution, and queues immutable
capture afterwards. Nested learned calls inherit the parent attempt. Domain code
can read `tasks.current()` and explicitly report advice decisions in
`signals.learning.advice`; runtime records remaining exposed candidates as
`unassessed`, not rejected or adopted. Optional `signals.learning.measurements`
accepts `first_action_at`, `tool_round_trips`, `visual_reasoning_calls`, and
`reused_workflow`. Unknown values stay absent; zero and false remain observations.
`signals.learning.observations` forwards up to ten explicit feedback observations
through the same frozen capture delivery.
Outcome mapping belongs to the domain contract and verified success requires
evidence. Transport success alone does not establish the task outcome.

For dynamic dispatch, use `lib.vrooli_memory.run_task(operation=..., inputs={...})`.
The receipt includes separate `outcome` and `delivery` plus an `attempt_id` and
`resume_token`. Keep the token private: it is the capability for later inspection
or capture recovery from a new session.

```python
receipt = lib.vrooli_memory.inspect_task(attempt_id=attempt_id, resume_token=token)
requeued = lib.vrooli_memory.resume_task(attempt_id=attempt_id, resume_token=token)
```

Recovery retries only frozen Memory write inputs with the pinned finish program.
It never repeats the domain action. An interrupted domain action remains unknown
until reconciled; starting another action is a new explicit attempt. Pending or
blocked delivery is not task failure and is not proof that a write did not commit.
Permanent input, authority, and immutable-record conflicts block delivery immediately;
transient transport failures retain bounded backoff. Partial acknowledgements are
retained. Recovery does not authorize replaying the domain operation.

## Reuse advice before acting

```python
choice = lib.vrooli_memory.choose_option(
    options=["flow-a@2", "flow-b@1"], default_id="flow-a@2").head(1)[0]
```

This read-only helper uses the current task's already-prepared advice; it does not
perform another recall. It recognizes only complete `option-preference/v1 ` JSON
entries with matching `operation` and `context_key` from `tasks.current()`, plus
an `option_id` in the caller's current allowed set. Conflicting, truncated, stale,
or free-form suggestions leave the fallback unchanged. The caller forwards
`signals.learning` and still owns authorization and outcome verification. Device
Control and Browser Automation Studio use it to recommend a listed workflow,
not to silently execute one. Applying a recommendation leaves its verdict unknown
until independent evidence establishes usefulness.

## Callable contracts

```python
prepared = lib.vrooli_memory.prepare_attempt(
    scope="owner-usage", task_id="stable-task-id", operation="inspect",
    context_key="target/profile/policy-v1", started_at="2026-09-01T12:00:00Z",
    task_started_at="2026-09-01T12:00:00Z", attempt_number=1,
    trigger="Inspect requested evidence", approach="Use bounded owner reads",
    provenance="operator", query="", advice_limit=5)

finished = lib.vrooli_memory.finish_attempt(
    scope="owner-usage", attempt=completed_attempt, observations=[])

comparison = lib.vrooli_memory.compare_outcomes(
    scope="owner-usage", operation="inspect", context_key="target/profile/policy-v1",
    cohort_limit=5, **{"from": "2026-09-01T00:00:00Z", "to": "2026-09-02T00:00:00Z"})
```

All arguments shown in preparation are required except `query` (empty derives
from operation/context/trigger) and `advice_limit` (5, accepted range 1–10).
Comparison requires only `scope`; `from`, `to`, `operation`, and `context_key`
default to empty strings. Comparison resolves the owner-style default of now and
the preceding seven days once, then sends both concrete timestamps to Memory
and returns them in envelope `inputs`. Replaying those `inputs` preserves the
exact window. The window is
half-open, historical, and at most 90 days. `from` is a Python keyword; pass it
through dictionary expansion. `cohort_limit` is 1–10, default 5.

Each child returns a Handle: read the envelope with `head(1)[0]`, inspect its
`status` and `errors`, and retain the child `meta()` artifact digest separately
when the caller needs execution identity. A successful runtime submission alone
does not imply successful child capture.

## Prepare: candidates are not advice uses

`signals` contains `attempt`, `recall_status`, `advice_candidates`,
`decision_required`, and `discarded_hits`. Candidates have `entry_id`, `text`
(at most 512 UTF-8 bytes), and `text_truncated`. The program reads only the
explicit scope, excludes summaries, duplicate IDs, and unusable hits, and
materializes at most `advice_limit` hits. It does not fetch more pages to replace
discarded hits. `no_match` means no usable candidate among these bounded hits.

The seed is a snake-case Memory Attempt without `finished_at` or `outcome`.
It contains a deterministic `attempt_id` and an empty `advice` array. An empty
successful read sets both retrieval and seed `recall_status` to `no_match`;
unavailable recall sets both to `unavailable`. A transport failure is explicit
in the envelope and does not prevent the caller from continuing its task.

Matched retrieval sets `signals.recall_status="matched"` and
`decision_required=true`, but leaves **the seed's `recall_status` absent**.
Before finish, the caller supplies `recall_status="matched"` and records:
`{entry_id, decision: applied|rejected|unassessed, decision_change,
verdict: supported|contradicted|unknown, evidence_refs}`. Retrieval alone does
not authorize inventing a decision, decision change, or supported verdict.
An unassessed exposure requires `verdict="unknown"`, empty `decision_change`,
and empty `evidence_refs`. Do not silently relabel matched
candidates as `no_match`; record an explicit rejection when that is what happened.
Runtime supplies unassessed exposures automatically; standalone helper callers
must supply them explicitly. Memory verifies advice IDs in the exact scope and checks
that the referenced entry predates the attempt start.

The caller then supplies the observed finish timestamp and outcome. Supported
task outcomes are `verified_success`, `failed`, `unavailable`, and `unknown`.
Verified success requires evidence references; failure requires a fingerprint.
Optional effort measurements stay absent when unknown. `0` and `false` are
observations and are never inserted as defaults.

## Finish: immutable writes and retry receipts

`attempt` and each observation use the existing proto's snake-case fields.
Finish accepts a complete Attempt without requiring preparation. IDs may be
explicit. Missing `attempt_id` is generated as `attempt-` plus SHA-256 of the
canonical JSON array `[scope, task_id, operation, context_key, attempt_number]`.
Preparation uses exactly the same algorithm. Use a task ID unique within its
scope and increment the ordinal for a new task execution, not for capture retry.

Observations are bounded to 10. Their `attempt_id` defaults to the resolved
attempt ID; a different ID is rejected. `disposition`, `provenance`, and
`observed_at` are explicit; no timestamp or feedback judgment is synthesized.
Supported dispositions are `supported`, `contradicted`, `insufficient`,
`unavailable`, `unresolved`, and `unknown`. All except `unknown` require evidence.
Missing observation IDs are `observation-` plus SHA-256 of canonical JSON
`[scope, observation_without_observation_id]`, after resolving its attempt ID.
Canonical JSON uses sorted keys, compact separators, ASCII escaping, and no NaN.
Identical generated observation bodies therefore have identical IDs; duplicate
IDs in one call are rejected. For separate identical feedback events, supply
distinct observation IDs. Keep IDs and bodies unchanged when retrying.

All nested inputs validate before any write. The complete resolved retry inputs
must fit 24,000 JSON bytes with ASCII escaping; oversized calls fail before writes.
Finish calls `learning.record` once and then `learning.observe` once per
observation, stopping at the first failure. All writes occur in the report phase.

The output shape is:

```json
{
  "signals": {
    "task_outcome": "verified_success",
    "capture_status": "complete",
    "entry_id": "attempt-receipt-id",
    "existing": false,
    "observations": [
      {"entry_id": "observation-receipt-id", "existing": false, "observation_id": "stable-id"}
    ],
    "retry_inputs": {"scope": "owner-usage", "attempt": {}, "observations": []}
  }
}
```

The abbreviated objects in that example contain the **complete exact write
bodies** in a real response. Reinvoke `finish_attempt(**retry_inputs)` unchanged;
the owner acknowledges identical existing writes. Do not change finished times,
IDs, outcomes, or observation bodies during retry. Changed bodies under an existing
ID surface the owner's immutable-record conflict. Corrections are new observations.

Capture failures return `status="partial"`, `capture_status="capture_failed"`,
and `errors[].class="capture_failed"` with `cause`, `cause_status`, and `where`
(`record` or `observe`). Successful receipts remain present. An unacknowledged
attempt has `entry_id=null`; `existing=false` then expresses no acknowledged
existing receipt, not proof of absence. A lost response may follow a committed
write. Governance refusal stops the call and remains visible in the cause.
Input failures return `failed/invalid_input` without writes or retry bodies.

Task outcome and capture status are independent: a failed task can capture
successfully, and verified task success can have partial capture. The programs
do not retry themselves. Printed output and session memory are not durable
checkpoints. Automatic learned calls use Program Runtime's durable checkpoint
and outbox outside these helpers. Standalone helper callers remain responsible
for retaining exact retry inputs and receipts if their workflow needs recovery.

## Compare: seven rows and reliability

`signals.rows` preserves the BAS/Knowledge Observatory board shape:
`{row, reading, target, in_band, unavailable, reason}`. The seven row names are
`failure-recurrence`, `completion-effort`, `advice-outcomes`,
`first-action-latency`, `agent-round-trips`, `visual-reasoning`, and `workflow-reuse`.
Each successful read carries `{from, to, eligible_attempts, cohorts}`;
each cohort retains full `operation`, `context`, and the existing camelCase metric
fields. Failure recurrence also includes `repeatedFailures`, and advice outcomes
includes `noMatch` and optional `contradictionRate`.

The owner computes cohorts; this program never combines distinct operations or
contexts or computes an inferred baseline. The caller compares windows by making
separate calls with identical operation/context selectors. Targets and in-band
judgments remain null. Optional metrics are null when absent; omitted nonoptional
proto counts are zero when the owner responded. Those count zeroes do not override
the response validity gate. An unreachable measure has null readings and unknown
denominators, with status `unavailable`; refusal remains `refused`.

`signals.reliability` preserves source reliability/reason, scan counters,
interpretation, total and returned cohort counts, and truncation. Multiple cohorts
remain separate and are not inherently unreliable. Too many cohorts return a
bounded sample with `unreliable:cohort_sample`. Scan truncation, invalid/legacy
records, missing validity, and an empty eligible set remain explicit. Unreliable
reads return `partial`; they cannot be treated as healthy observations. If full
identities would exceed the output budget, readings become null with
`unreliable:output_bound`; identities are never truncated into apparent matches.

Learning comparisons retain operator, test, and agent provenance in cohort
identity and expose per-provenance attempt counts in `signals.reliability`.

## Focused validation

```sh
python3 -B -m unittest discover -s scenarios/vrooli-memory/.vrooli/program-runtime/tests -p test_learning_programs.py -v
```

The tests use fake bounded Handles and immutable-ID writes, including disconnect
after commit, exact replay, partial observations, conflicts, refusal, no-match,
unavailable reads, multiple cohorts, optional zero versus unknown, and output
bounds. These tests do not establish live bridge or owner behavior. The main
integration tests separately establish bridge, owner, and restart behavior.
