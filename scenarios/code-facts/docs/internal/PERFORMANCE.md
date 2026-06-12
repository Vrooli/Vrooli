# Performance — Code Facts

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
| Repeated selective describe | Provider extraction avoided after identical cached request | API unit test with counting provider | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Repeated `DescribeCodeFacts` cache reuse | second identical request uses report cache and makes zero provider calls | `internal/facts` unit test | 2026-06-12 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Cache fingerprinting walks bounded parse-unit roots and prunes dependency/build directories.
- Performance budgets for proof synthesis should be revisited after Phase 11 exposes larger operator UI workflows and Phase 12 adds the first external consumer.

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
