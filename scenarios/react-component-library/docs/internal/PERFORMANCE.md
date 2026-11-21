# Performance — React Component Library

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](TESTING.md#behavior-claim-register).

This document records performance budgets, current measurements, known
constraints, and regression procedures.


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
| None captured yet. | n/a | n/a | 2026-05-12 |

## Workbench redesign budgets (planned)

These are targets, not current measurements:

| Surface | Target | Why |
|---|---|---|
| Fixture drag | 60 fps pointer movement; no per-frame React state update for all fixtures | A canvas that rerenders every card per frame will feel worse than the current grid |
| Home / Components first paint | No synthetic blocking work; real projections stream or show explicit loading | The current catalog already exposes pending previews |
| Drawer open/close | 120–200 ms transition, no layout thrash | The drawer is a frequent evidence surface |
| Design artboard resize | Stable geometry during pointer move; no per-frame full page rerender | Resize must not drop the selected component or reset scroll |
| Coverage / Review queues | Bounded render window; no unbounded DOM for findings | The queue can grow beyond the sample |

If a target cannot be met, record the measured cost and reduce the surface rather than hiding a spinner over a broken interaction. See [WORKBENCH-REDESIGN.md](../concepts/WORKBENCH-REDESIGN.md).

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
