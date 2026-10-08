# Performance — Persona

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Performance Character

This scenario is **latency-shaped, not throughput-shaped**. It handles a
handful of personas, a small queue, and occasional bursts of activity
when a flow is running. No table is expected to exceed low tens of
thousands of rows except the journal, which grows monotonically and is
managed by export-and-archive rather than deletion.

The consequence: local query performance is a non-issue, and every
meaningful budget below is dominated by **something other than this
scenario's own code** — a live verification call, a mailbox poll, or a
human. Optimising SQLite access here would be optimising the wrong
thing.

## Budgets

| Operation | Budget | Dominated by | Notes |
|---|---|---|---|
| Persona read / list | < 50 ms p95 | Local SQLite | Pure local read; a regression here indicates a query defect, not load. |
| Act-as authorization | < 400 ms p95 | **The live `agent-manager` verification call** | The scenario's own work is a few milliseconds. Budget exists to detect verification-path degradation, not local slowness. |
| Entitlement resolution | < 100 ms p95 | Local, plus an optional `prompt-manager` grant read | Falls back to local ACL when grants are unavailable. |
| Handoff open / complete | < 150 ms p95 | Local write plus journal append | Excludes delivery, which is asynchronous. |
| Handoff delivery attempt | < 2 s p95 | `notification-hub` | Asynchronous; never blocks handoff creation. |
| Code retrieval | **No latency budget** | The counterparty and the mailbox | Bounded by the caller's window, not by a target. A timeout is a named outcome, not a performance failure. |
| Document release | < 1 s p95 | `document-manager` round trip | Idempotent per (binding, handoff), so a retry is cheap. |
| Journal query (filtered) | < 200 ms p95 | Local SQLite with indexes | Watch as the journal grows; the first place growth will show. |
| Journal export | < 30 s for 1M rows | Streaming write | Must stream, never buffer the whole journal in memory. |

**The number that actually matters is not in this table**:
time-to-completion on a handoff, which is dominated by a human and
measured in minutes to hours. It is a product metric rather than a
performance budget, and it is currently unmeasured — see
[`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md).

## Current Measurements

**None.** No implementation exists, so every budget above is a target
derived from the dependency shape rather than an observation. Populate
this section from real runs once the first vertical slice lands; do not
treat the budgets as validated until then.

| Operation | Measured | Date | Conditions |
|---|---|---|---|
| _Nothing measured yet._ | — | — | Scenario is documentation-complete and implementation-empty. |

## Known Constraints

- **Verification is a network call on the hot path.** Act-as cannot be
  cached, because a cached authorization is exactly the failure mode the
  fail-closed rule exists to prevent. This is a deliberate latency cost.
- **The journal only grows.** Append-only is a correctness guarantee, so
  the mitigation is export-and-archive, never pruning. Growth is slow
  (one row per meaningful action) but unbounded.
- **Handoff checkpoints are variable-size.** A checkpoint embedding a
  large mid-enrolment form is the one row shape that could get big.
  Retention is short partly for this reason.
- **Code retrieval polls.** Polling a mailbox within a caller's window
  is inherently latency-bound by the counterparty; no local optimisation
  changes it.
- **SQLite serialises writes.** Correct and sufficient here, because
  writes are agent-initiated and serialised by the requesting flow
  anyway. This assumption is recorded in
  [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) and should
  be revisited only if concurrent external writers ever appear.

## Regression Procedure

1. Reproduce with the scenario's own test suite:
   `vrooli scenario test persona`.
2. **Separate local from dependency latency before investigating.** Act-as
   and release budgets are dominated by other scenarios; a regression is
   far more likely to be theirs. Check `act_as_latency` against
   `dependency_reachable` first.
3. For local regressions, profile the query — an unindexed journal
   filter is the most probable cause as the journal grows.
4. Record the measurement in Current Measurements above, with date and
   conditions, whether or not the cause is found.
5. If a budget proves wrong rather than the code, change the budget
   here and note why in [`DECISIONS.md`](DECISIONS.md). A budget nobody
   believes is worse than no budget.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
