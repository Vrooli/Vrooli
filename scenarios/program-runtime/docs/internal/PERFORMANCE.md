# Performance — Program Runtime

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
| Per-session inference spend | configured cost and token ceilings from ai-gateway `Usage` | session budget ledger; `cost_micros`, input tokens, output tokens | planned (`OT-P1-010`) |
| Per-session delegated-run spend | separate configured ceiling for agent-manager work | delegated-run usage ledger; reclamation/refusal reason | complete (`OT-P1-011`) |
| Agent-facing program result | constant bounded response, independent of source result cardinality | real-kernel scaling test | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| `agent_bytes` for 10, 1,000, 100,000, 10,000,000 bounded rows | 16, 16, 16, 16 bytes; full materialization at 10,000,000 rows: 65,536 bytes | `api/internal/programs/scaling_test.go` real subprocess | 2026-08-11 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Performance budgets for real product workflows must be defined after
  domains and UX flows are known.
- Inference and delegated-run budgets are intentionally separate. A typed
  inference call is bounded by ai-gateway usage, while a delegated run is an
  agentic cost tier with different pricing and failure semantics. One shared
  ceiling would make either budget unenforceable or unnecessarily restrictive.

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
