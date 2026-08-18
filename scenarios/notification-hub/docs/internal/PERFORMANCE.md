# Performance — Notification Hub

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |
| Send request acceptance | p95 under 100ms | API timing on the send endpoint | planned with OT-P0-002 |
| Routing decision | p95 under 10ms | Unit benchmark; the decision is a pure function over cached reads | planned with OT-P0-005 |
| Accept to first delivery attempt | p95 under 5s for non-held notifications | `notifications` and `deliveries` timestamps | planned with OT-P0-004 |
| Accept to delivered | p95 under 30s, local channels | Same | planned with OT-P0-004 |
| Timeline query | p95 under 200ms over a full retention window | API timing on the list endpoint | planned with OT-P0-004 |
| Relayed delivery overhead | under 5s added versus local, node online | Delivery timestamps by path | planned with OT-P1-001 |

**Throughput is deliberately not a budget.** The expected load is tens
of notifications a day for one owner, not thousands a second. The
previous implementation's stated ~100/second ceiling was never the
constraint that mattered; delivering zero of them was. Latency and
correctness budgets are listed here because they affect whether a
notification is useful when it arrives; throughput would be a number
tracked for its own sake.

If throughput ever does become the constraint, that is the revisit
trigger for the in-process queue decision in
[`DECISIONS.md`](DECISIONS.md) — and the evidence must be a measurement,
not an intuition.

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-08-17 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Performance budgets for real product workflows must be defined after
  domains and UX flows are known.

## Regression Procedure

1. Run `make test`.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
