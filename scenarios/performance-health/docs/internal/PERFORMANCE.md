# Performance — Performance Health

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
| None captured yet. | n/a | n/a | 2026-06-21 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Interaction budgets for real product workflows can be declared per flow under
  `performance.budgets.flows.<slug>` in `.vrooli/testing.json`. Gesture flows
  should budget frame health (`drawn_fps_min`, `dropped_frame_rate_max`),
  browser work (`raster_total_max_ms`, `layout_total_max_ms`,
  `paint_total_max_ms`), long tasks, and input evidence
  (`input_event_count_min`) in addition to LCP and component commits. Missing
  frame/input evidence fails closed unless the flow is explicitly `load_only`.

## Regression Procedure

1. Run `make test`.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, author a perf flow
   (`scenarios/<scenario>/bas/flows/<slug>.json`, `intent: performance`) and
   capture it with `performance-health audit run <scenario> --workflow <slug>`
   (see the `performance` steer skill). The old hand-edited
   `ui/perf/capture.template.js` is retired — this is its productized superset.
4. Set or check a flow budget with `performance-health budget set <scenario>
   --flow <slug> ...` and `performance-health budget check <scenario> --flow
   <slug>`. Budget breaches fail the test-genie Performance phase and therefore
   the suite run, not `git-control-tower baseline diff`.
5. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
