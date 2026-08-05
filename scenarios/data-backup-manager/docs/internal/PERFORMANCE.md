# Performance — Data Backup Manager

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
| UI Lighthouse performance | `>= 0.75` error floor, `>= 0.85` warning floor | Test Genie performance phase, desktop simulated Lighthouse audit | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Lighthouse performance | `0.71` (below error floor) | Initial comprehensive run; preserved as incident history | 2026-08-05 |
| Lighthouse performance | Passed configured floor; performance phase clean | Final comprehensive run `20260805-213135-0bcf84b3` | 2026-08-05 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- The initial Lighthouse score was below the configured floor; later overview
  rendering and route/bundle work produced a clean performance phase. The
  earlier score remains historical evidence and must not be hidden by lowering
  the threshold.

## Regression Procedure

1. Run `make test` (or the owned comprehensive Test Genie run).
2. Inspect the performance phase artifact and Lighthouse report for the run.
3. Capture relevant API/UI command timing.
4. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
5. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
