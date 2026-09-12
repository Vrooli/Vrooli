# Performance — Token Economy

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

**Volume here is inherently tiny.** A household produces on the order of tens of
events per week, not thousands per second. Almost nothing in this scenario is
performance-sensitive, and pretending otherwise would produce budgets nobody
checks. Two things genuinely matter: the holder view must feel instant on a
phone, and the projection must not become the reason a balance is wrong.

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |
| Holder view first meaningful paint | < 1.5s on a mid-range phone over local network | BAS capture at mobile viewport | planned |
| Balance read (projection hit) | < 50ms p95 | API handler timing | planned |
| Balance read (projection miss, full replay) | < 500ms p95 at 10k events | `api/internal/journal` benchmark | planned |
| Redemption settlement (debit + event, one transaction) | < 100ms p95 | API handler timing | planned |
| Projection rebuild, full | < 5s at 100k events | maintenance command timing | planned |
| Approval queue load | < 200ms p95 | API handler timing | planned |

**Why the projection budgets are the ones with teeth.** Balance is a query over
an append-only journal that is never compacted (`TKE-P0-004`). The journal grows
monotonically for the life of the household. If projection reads ever become
slow enough that someone is tempted to trust a stored total instead, the
scenario's central correctness property is what gets traded away. The budget
exists to make that pressure visible early rather than at the point of
temptation.

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured. | n/a | Nothing is implemented; every budget above is a target, not an observation. | 2026-08-18 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes. Inherited from the template; accepted.
- **SQLite is single-writer.** Every mutation takes a row lock, so writes
  serialize. This is correct and sufficient for a household, where concurrent
  writes are effectively nonexistent. It is also the one thing that would not
  survive `TKE-P2-001` (a real-value rail with concurrent external payers) —
  recorded as the single migration trigger in [`../concepts/DATA.md`](../concepts/DATA.md).
- **The journal never shrinks.** No compaction, by design (`TKE-P0-010`).
  Growth is bounded by household volume, which is small, but projection cost
  scales with history and must be measured against a realistic multi-year
  event count rather than a fresh database.
- **Rule evaluation is deterministic and local.** No inference, no network call,
  no external dependency in the hot path — a deliberate design decision recorded
  in [`DECISIONS.md`](DECISIONS.md), and the reason redemption latency is a
  storage question rather than a provider question.
- **The holder view is a phone surface first.** Its budget is about perceived
  responsiveness on a mid-range device over a home network, not about server
  throughput.

## Regression Procedure

1. Run `make test`.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. **For any balance or settlement regression, benchmark against a seeded
   journal of at least 10k events, never a fresh database.** A projection
   regression is invisible at small history and is exactly the class of problem
   that erodes into "just store the total".
5. If a projection read regresses, fix the projection — never introduce an
   authoritative stored balance. That trade is explicitly closed in
   [`DECISIONS.md`](DECISIONS.md).
6. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
