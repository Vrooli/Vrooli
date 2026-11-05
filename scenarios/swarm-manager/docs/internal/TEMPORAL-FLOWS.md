# Temporal Flows Documentation

## Declared workflow lifecycle

1. An operator or deterministic domain event selects a transition from the
   registry.
2. Swarm Manager authorizes the request, preflights integrations, and builds an
   immutable input snapshot with an idempotency key.
3. Agent Manager starts a pinned declared workflow revision and records its
   journal.
4. Swarm Manager collects a typed terminal result, verifies that its captured
   domain frontiers are current, and applies it exactly once.
5. A stale, rejected, cancelled, or abstained result is preserved as history
   without a domain mutation.

Workflow waits, branches, retries, child work, and budgets are durable Agent
Manager behavior. Swarm does not poll or reconstruct those mechanics.

## Lost-callback reconciliation

Swarm stores the workflow correlation, not a second copy of workflow progress.
Agent Manager is the authoritative reader. At startup and on each background
cycle, Swarm examines only inspectable execution records and reads the
correlated workflow's durable terminal state:

```text
execution record (starting|running|needs_review)
        │  OpWorkflowID, or transition correlation fallback
        ▼
Agent Manager trace (read-only)
        │
        ├─ non-terminal ───────────────▶ leave record unchanged
        └─ terminal
             ├─ succeeded + result evidence ─▶ completed
             ├─ succeeded without evidence ─▶ needs_review
             ├─ blocked/abstained/budget ───▶ needs_review
             ├─ failed ────────────────────▶ failed
             └─ cancelled ─────────────────▶ canceled
```

The transition is idempotent: the record is saved before notifications or
backlog projection, and a repeated sweep recognizes the already-applied
terminal reason. A successful workflow without terminal result evidence is
fail-closed and cannot silently mark domain work complete. Diagnostics should
include execution id, workflow id, terminal code, workflow updated-at, record
status, and the next action (review, retry, or inspect the Agent Manager
trace).

| State | Owner | What a restart does | Human action |
| --- | --- | --- | --- |
| `starting`/`running` | Agent Manager workflow + Swarm projection | read trace and reconcile if terminal | wait or inspect trace |
| `needs_review` | Swarm domain projection | re-read terminal evidence; never auto-apply a missing result | review/apply explicitly |
| `completed`/`failed`/`canceled` | Swarm durable record | no duplicate terminal transition | inspect history or retry |

This sweep is a recovery backstop, not a consumer-facing polling API. Managed
waits use Agent Manager's server-owned wait; human shell waits may remain
blocking and are not used as a substitute for durable workflow reconciliation.

### Recovery runbook and troubleshooting

1. Capture the Swarm execution id, correlated workflow id, current execution
   status, workflow terminal code, workflow `updated_at`, and the next action.
2. Read the Agent Manager workflow trace by workflow id. Confirm terminal
   result evidence exists before treating success as completion.
3. Let the startup/periodic reconciliation sweep apply the mapping. Do not
   create a new workflow or mutate the execution row by hand.
4. If the record becomes `needs_review`, review the terminal result or its
   absence and apply/retry explicitly. `failed` and `canceled` remain visible
   terminal outcomes with their original correlation.

Common symptoms map to ownership boundaries:

- Workflow terminal, execution still running: inspect the read-only trace
  reader and reconciliation diagnostics; the callback path was lost.
- Success without evidence: expected fail-closed behavior; inspect the
  workflow result contract rather than marking the execution complete.
- Duplicate terminal notifications: expected to be harmless; the durable
  execution projection and completion/finalization path are idempotent.
- Human shell wait still blocking: that is a human-owned wait and is not an
  Agent Manager parked run. Managed agents must use the park/wake contract.

## Human session lifecycle

An Agent Session is created before its first message. Operator messages may
continue that session's Run, while session context and proposals are persisted
by Swarm Manager. Historical sessions remain viewable; retired kinds cannot be
created or resumed.

## Recovery

Workflow correlations and application claims make process restart safe. A
terminal result may be collected again after a crash, but the existing claim is
returned rather than applying the result twice. Integration outages defer or
reject starts according to the transition's declared requirements.
