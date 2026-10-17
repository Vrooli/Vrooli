# Performance — Device Control

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

### Scenario-specific budgets

Device work is dominated by physical latency, so the budgets that matter are
about *not adding* avoidable cost.

| Surface | Budget | Why |
|---|---|---|
| Capability probe | Cheap enough to run on a schedule without disturbing an idle device | A probe that wakes a phone every minute is a battery bug, and the user notices before we do. |
| Target resolution | Rung-dependent by design: `semantic` and `visual-anchor` are local and effectively free; `vision` costs an `ai-gateway` round trip plus tokens | The ladder is partly a performance mechanism — the fast path should be taken whenever the strategy allows it. |
| Bounded waits | Explicit upper bounds, always | The performance question is never "how long did it sleep" but "was the bound appropriate and was it exceeded." |
| Frame streaming | Degrades to a lower frame rate rather than blocking verb dispatch | Control latency is the property that matters; the frame is evidence. |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-08-10 |

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
