# Performance — Search Hub

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
| Federated fan-out | `ceil(active providers / concurrency) × provider timeout < query timeout` | deterministic budget invariant test | active |
| Address resolution | one lookup per scenario/port within the cache TTL | resolver fake-clock/cache tests | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Query budget | 34 active leaves at concurrency 8 and 4s provider timeout fit in 20s of a 25s query budget | `internal/routing/fanout_budget_test.go` | 2026-08-12 |
| Address cache | successful entries live for 2s by default; failures invalidate immediately | `packages/api-core/discovery` fake-clock tests | 2026-08-12 |
| Live p50/p95 | 7,235 / 20,203 ms | `search-hub metrics federated-latency --window 7 --json` | 2026-08-12 |
| Degraded-query rate | 334 / 1,059 (31.54%) | `search-hub metrics degraded-query-rate --window 7 --json` | 2026-08-12 |
| Federation state | 31 providers; 0 demoted; 1 quality-withheld | `search-hub federation status --json` | 2026-08-12 |

## Honest Comparison

The three columns below preserve the audit, restored-runtime baseline, and
current snapshot. They are directional evidence, not a claim of a clean
single-variable benchmark: the final snapshot includes later owner tuning,
expanded suite registration, and the current degraded resource state.

| Signal | Pre-restoration audit | Honest baseline | Final/current snapshot | Direction and reason |
|---|---:|---:|---:|---|
| Answer coverage | 17% (6/36) | 25% (9/36) | 22.22% (8/36) | Down from the restored baseline; current coverage is the live NOW projection and remains denominator-confident only for its declared model. |
| Demoted providers | 18/34 | 0 observed | 0/31 | Improved and held; restart plus successful probes cleared transport-era demotions. |
| Aggregate eval met | 0/235 | 191/235 (222 graded) | 190/237 (222 graded) | Slightly down in met count, but not a like-for-like denominator: two suites were added and the graded denominator stayed constant. |
| 7-day p95 latency | 19,594 ms | 20,203 ms | 20,203 ms | Flat versus the restored baseline; no latency win is claimed while the provider fleet remains degraded. |
| 7-day zero/degraded rate | 31% | 31.51% (328/1,041) | 31.54% (334/1,059) | Flat within measurement drift; the additional queries are retained rather than hidden. |
| Blocking maturity findings | 23 | 18 | 19 | Improved versus the phase-1 audit and remains below the `<23` certification threshold; the remaining findings are recorded debt, not hidden. |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Live before/after p50 and p95 comparisons require the honest baseline with
  qdrant and ollama healthy. The current values are recorded above, but the
  shared runtime remains degraded, so no latency improvement is claimed.

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
