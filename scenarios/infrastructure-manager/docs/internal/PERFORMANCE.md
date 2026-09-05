# Performance — Infrastructure Manager

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
| Per-source read | **3s deadline** (`readDeadline`) | per-client instrumentation | planned |
| Setpoint + space read | **10s deadline per owner** | `coverage` domain | planned |
| Board response (`focus next`) | **under 10s** with all sources reachable | end-to-end handler timing | planned |

**A slow source must become an honest `UNAVAILABLE`, never a hang.** That is the
performance budget that actually matters here, and it is a correctness property
rather than a nicety: a board that blocks on one unreachable dependency is
indistinguishable from a board that is down, and the team loses its only address
at exactly the moment the platform is misbehaving. Sources are read
concurrently, each bounded independently, and a source that misses its deadline
degrades to a stated-reason entry while the rest keep ranking.

These deadlines mirror `meta-optimization-manager`'s `numeratorDeadline = 3s`
and `spaceReadTimeout = 5s`. They are named constants so the judgment is
auditable, per the load-bearing-constants discipline in
[`../concepts/SETPOINT-MODEL.md`](../concepts/SETPOINT-MODEL.md).

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| UI production build and typed API unit suites | passing | build and suite timings recorded by the scenario-owned validation run | 2026-08-20 |

One upstream measurement is worth recording because it will shape the read
budget: every test-genie phase-batch admission was once observed running a full
host inventory at ~8.2s on this host. Any source whose read path can trigger
host inventory needs its deadline verified against that behaviour rather than
assumed.

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- **Reads fan out to as many as seven dependencies**, several of which are
  themselves supervised elements that may be slow or down precisely when the
  board is most needed. Concurrency plus per-source deadlines is a structural
  requirement, not an optimization.
- **Reading history grows without bound unless retention is enforced**, and the
  retention floor is set by the longest declared target window (30d today, so
  45d with margin). Trimming below that floor silently converts a measurable
  target into an unmeasurable one — see [`../concepts/DATA.md`](../concepts/DATA.md)
  § Retention And Deletion. Storage pressure is never a valid reason to trim
  below it.
- Band evaluation is recomputed at query time rather than stored, so query cost
  scales with history depth. If that becomes the bottleneck, the fix is
  narrowing the queried window — never persisting verdicts.

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
