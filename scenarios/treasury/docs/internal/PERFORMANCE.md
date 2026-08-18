# Performance — Treasury

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
| Authorization decision | under 150 ms p95, excluding the identity round-trip | API timing on `AgentSpend.Authorize` | planned |
| Identity verification round-trip | under 300 ms p95 | Timing of the `agent-manager` verification call | planned |
| Settlement commit | under 100 ms p95 for the local transaction, excluding the rail call | API timing on the settlement mutation | planned |
| Approval queue first paint | under 1 s at the desktop baseline | Experience capture | planned |
| Inbound x402 admission | no lock-timeout failures under concurrent payers | `TRS-P1-002` performance validation | planned |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-08-18 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- **The identity round-trip is on the critical path of every
  authorization.** Verification is a live call to `agent-manager` and is
  deliberately not cached, because a cache converts a revocation into a
  delayed refusal. Authorization latency therefore has a floor set by
  another scenario, and the authorization budget above excludes it for
  that reason: the number this scenario controls should be measured
  separately from the number it does not.
- **SQLite serialises writers.** For single-operator authorization volume
  this is a feature — it satisfies the row-locking discipline directly.
  The one place it could bind is inbound x402 metering, where many
  external callers may pay concurrently. That is the single declared
  storage-migration trigger; see
  [`../concepts/DATA.md`](../concepts/DATA.md).
- **Headroom is computed rather than stored**, by decision. The cost grows
  with authorization history per budget. At single-operator volume this is
  immaterial; if it stops being immaterial, the answer is a derived
  read-model with its own invalidation, not a mutable balance field.
- **Evidence is append-only with no purge**, so storage grows
  monotonically with authorization volume. Bounded by construction at
  single-operator scale.
- Performance work should not begin before implementation. Every budget
  above is a target, not a measurement.

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
