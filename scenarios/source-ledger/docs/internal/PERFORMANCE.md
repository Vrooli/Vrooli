# Performance — Source Ledger

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

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Warm recall p95 (5 sequential runs) | 1.16 s | `vrooli-memory recall recall "durable shared memory and scope" --scope agent-memory --limit 5 --json`; steady samples 1.12–1.16 s after a 1.87 s cold sample | 2026-08-07 |
| Warm wake p95 (5 sequential runs) | 0.93 s | `vrooli-memory wake --scope agent-memory --budget 120 --json`; samples 0.85–0.93 s | 2026-08-07 |

These are the pre-extraction baselines against the live managed `vrooli-memory`
scenario after the compaction canopy was restored. They are not fixture or
empty-forest measurements.

## Post-Extraction Ceilings

| Surface | Ceiling | Rationale |
|---|---:|---|
| Warm wake p95 | 1.20 s | Allows a bounded cached discovery/network hop over the 0.93 s baseline. |
| Warm recall p95 | 1.50 s | Allows a bounded cached discovery/network hop over the 1.16 s baseline. |

The ceiling is a regression gate, not a promise that the first cold request is
warm. Cold-start and provider enrichment timings remain separately observable.

## Known Constraints

- Vite production builds may process the generated module graph and take
  several minutes.
- Performance budgets for real product workflows must be defined after
  domains and UX flows are known.

## Regression Procedure

1. Run the server-owned comprehensive suite and record its run id.
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
