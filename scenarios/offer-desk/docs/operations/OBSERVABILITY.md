# Observability — Offer Desk

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## The rule that shapes this document

Two PRD targets are observability requirements written as product requirements:

- `OT-P1-003` — the ranked board's sources must **degrade independently**. That is only
  provable if each source's availability is separately observable.
- `OT-P1-002` — an unavailable ledger is stated with a reason and **never reported as
  zero**. Same shape as Money Ledger's honesty rule, and it fails the same way if the
  signals do not exist.

A third property is specific to this scenario: **scheduled evaluation is invisible when it
works and invisible when it silently stops.** A trigger evaluator that quietly dies
reproduces exactly the failure this scenario was built to end — candidates asleep forever —
except now with a green dashboard. Evaluation liveness is therefore the single most
important signal here.

The corollary constraint: revenue figures read from Money Ledger pass through this scenario
and are never stored. **They must not be logged or used as metric labels either**, or
observability becomes the persistence layer `SECURITY.md` forbids.

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |
| **Evaluation liveness** | health | `gates` | Time since the last completed evaluation run | must be within one scheduled interval; exceeding it is the scenario's primary alarm |
| Evaluation outcome mix | correctness | `gates` | Per run: triggers evaluated, fired, and **unknown-blocked** | a rising unknown-blocked count means the fact registry is decaying |
| Candidates without a parseable trigger | integrity | `catalog` + `gates` | Direct measure of the `OT-P0-003` invariant | must be zero; non-zero means enforcement has a hole |
| Refused transitions | product | `catalog` | Count by rule that refused | healthy and expected; a refusal is the system working |
| Ledger availability | health | `board` | Whether the actuals join could be read, with reason | unavailable is surfaced, never rendered as zero |
| Per-source board degradation | health | `board` | Which board sources contributed to the current ranking | each source independently reported |
| Proposal age | product | `gates` | Time from proposal creation to operator disposition | rising age means the operator boundary has become a bottleneck |

**`unknown-blocked` is the signal to watch.** A trigger that cannot evaluate because a fact
is missing is neither a pass nor a fail, and it is the state most likely to accumulate
silently. It is separately counted for exactly that reason.

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses the deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Evaluation run log | `gates` | `make logs` | One entry per run: start, triggers evaluated, fired, unknown-blocked, duration. |
| Refusal log | `catalog` | `make logs` | Node, attempted transition, rule that refused. The refusal reason is the product; log it in full. |
| Import run log | `catalog` | `make logs` | Per-source-file counts. The verification evidence for `OT-P0-006`; retain it past the source deletion. |
| Audit trail | `catalog` (database, not a log file) | CLI query | Append-only durable record of actor, timestamp, prior value, reason. Not a log stream. |

**Prohibited in logs:** revenue amounts read from Money Ledger, and any fact value that is
itself a commercial figure. Log that a fact was resolved and whether it satisfied the
predicate — never the number.

## Metrics

| Metric | Status | Details |
|---|---|---|
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Time since last evaluation run | **required** | The scenario's most important metric. Backs the liveness alert below. |
| Triggers fired per run | required | Label by trigger id, never by fact value. |
| Unknown-blocked count | required | Fact-registry decay detector. |
| Candidates lacking a trigger | required | Should be structurally impossible; measuring it proves enforcement rather than assuming it. |
| Refused transitions by rule | required | Also a UX signal: one rule refusing constantly means the rule or the workflow is wrong. |
| Ledger read availability and latency | required | Backs `OT-P1-002`. Short deadline means latency matters. |
| Proposal age distribution | recommended | Measures the operator boundary's health, not the system's. |
| Board composition | recommended | Which sources contributed. Proves independent degradation actually happened rather than being designed for. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. The board computes at read time across several sources, so its latency is a real budget. |
| Cost telemetry | not-applicable | No gateway usage, no per-user infrastructure. |

## Alerts / Health

- **Evaluation liveness is the one real alert.** If the scheduler has not completed a run
  within its interval, the scenario has silently reverted to the behaviour it replaced.
  This is the only condition here worth interrupting a human for.
- **A refused transition is not an alert.** It is the system working as designed and
  belongs in the response to the caller, not in an operator's attention.
- **An unavailable ledger is a user-facing state, not an alert.** The board surfaces it as
  an availability entry while healthy sources continue to rank.
- **A fired trigger is a notification, not an alert.** It produces a proposal for the
  operator to disposition on their own schedule — the scenario exists so that this stops
  being urgent.

## What "users are getting value" looks like here

| Question | Signal | Notes |
|---|---|---|
| Did the scenario do something prose could not? | Count of triggers fired unassisted | `trigger-met` was never once reached in the source documents. The first unassisted firing is the proof asset named in `GO-TO-MARKET.md`. |
| Is the catalog honest? | Candidates lacking a trigger stays zero | The invariant that made the old catalog decay. |
| Is the operator boundary working or blocking? | Proposal age distribution | Proposals dispositioned means the loop closes; proposals piling up means the boundary has become the new bottleneck. |
| Is anything active and earning nothing? | Board entries of that class | The highest-value output of the Money Ledger pairing, and the reason both scenarios exist. |

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| **No evaluation-liveness signal yet** | A dead scheduler is indistinguishable from a quiet one, silently restoring the original defect. | Build with the `gates` domain, in the same slice as the scheduler. |
| No unknown-blocked counter | Fact-registry decay is invisible until candidates have been stuck for months. | Same slice as trigger evaluation. |
| Import verification evidence not durable | `OT-P0-006` requires verified counts before source deletion; if the evidence is only in a log that rotates, the verification cannot be re-checked. | Before the import runs. Retain the import log as an artifact. |
| No agent-vs-operator attribution in metrics | Cannot measure whether the promotion boundary is respected in practice. | Blocked on the identity gap in `SECURITY.md`; same trigger. |
| Product usage telemetry | Cannot validate the standalone hypothesis. | Only if the `MONETIZATION.md` revisit trigger fires. Not needed for the committed internal role. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — what must never be emitted
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
