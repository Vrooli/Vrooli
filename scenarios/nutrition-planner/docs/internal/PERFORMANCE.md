# Performance — Nutrition Planner

This document records performance budgets, current measurements, known
constraints, and regression procedures.

The working product name is **Daily**. Scenario id is `nutrition-planner`.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

Performance here has two faces. The **deterministic core** (nutrition
arithmetic, recipe scaling, candidate filtering, plan evaluation) must be
fast and reproducible; it is pure and can run without a browser, database,
network, or model (`ARCH-02`). The **console** must feel immediate for
local controls and must not freeze manual editing while optional AI or
provider work runs.

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| Foreground plan search | **~2 seconds** on the reference development environment, as a tunable search budget (`PLN-03`) | Test Genie performance phase / benchmark harness | declared, not measured |
| Local control feedback | Immediate — slider, checkbox, and editor keystrokes update a local preview without a server round-trip (`ARCH-03`) | UI interaction capture | declared, not measured |
| First useful screen | Render without waiting for AI or a provider; manual data is sufficient | UI load timing | declared, not measured |
| Long exports / imports | Shifted to progress-visible background jobs with cancellation (`JOB-01`, `OPS-01`) | Job duration metric | declared, not measured |
| API health | Responsive under the lifecycle health-check timeout (`/health`) | lifecycle health check | active (scaffold) |
| UI health | Responsive under the lifecycle health-check timeout (`/health`) | lifecycle health check | active (scaffold) |
| UI build | 5–10 minutes accepted for the current Vite module graph | lifecycle/test-genie build logs | inherited |

The search budget is a **budget, not a promise**: a slow or timed-out
search returns its best validated draft plus a search status. A timeout
means "no feasible plan found in this search," never proof that none exists
(`PLN-06`, `ACT-047`). The numbers `20` candidates per slot, `8` branching,
beam width `~24`, and the ~2s budget are recommended configurable defaults
to be benchmarked, not hardcoded promises (`PLN-03`).

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-09-18 |
| Product domain not implemented, so no real workflow exists to measure. | n/a | — | 2026-09-18 |

The only currently measurable performance is the generated scaffold's
build and health behavior. Do not record scaffold timings as product
performance.

**Reference dataset.** Measure against the declared reference dataset
rather than whatever data happens to be on hand (spec §19.1). These are
test sizes, not product limits:

| Dimension | Size |
|---|---|
| Saved recipe revisions | 1,000 |
| Days of plans and intake history | 90 |
| Products | 200 |
| Planning request | one seven-day, full-day plan |

The reference environment is the project's standard development
environment, stated alongside any measurement so a number cannot be
compared across unlike machines.

## Known Constraints

- The **Vite production build** may process thousands of modules and take
  several minutes; this is a build-time cost, not an interaction budget.
- Deciding what to eat must stay fast even when optional work is slow. A
  slow provider must not freeze manual editing, and expensive work needs
  cancellation and a visible status with partial results clearly identified
  (`OPS-01`).
- Recalculating an entire history for a single checkbox is a known
  anti-pattern: paginate/search large collections, cache versioned
  assessments, and batch provider calls where permitted (`OPS-01`,
  `ARCH-03`).
- **Use virtualization only where a measured list size warrants it**, and
  preserve keyboard access and stable focus (`OPS-01`, `UX-VIS-03`).
- Planning is plan-level, not dinner-at-a-time, once nutrients, package
  costs, leftovers, and repetition interact; this is why the search needs a
  bounded budget rather than unbounded computation (`PLN-03`).
- No measurements exist yet, so every product budget above is a target, not
  a baseline.

## Regression Procedure

1. Run the performance phase through Test Genie, which owns the run:
   `vrooli scenario test nutrition-planner --phases performance` (or the
   `test-genie.iterate` program). The run is server-owned and survives a
   cancel; wait once with
   `test-genie runs wait --json nutrition-planner <run-id>`.
2. Seed the reference dataset above before measuring; state the environment
   and dataset size with every result.
3. Measure the deterministic core directly (search duration, evaluation
   duration, arithmetic throughput) and the console separately (control
   responsiveness, first useful screen, list interactions at scale).
4. For UI interaction regressions, follow the capture template under
   `ui/perf/` if present; otherwise capture the equivalent DevTools trace
   and attach it to the result.
5. Record a persistent finding here if it is an accepted constraint, or in
   [`PROBLEMS.md`](PROBLEMS.md) if it is unresolved debt. A budget change is
   a durable decision and belongs in [`DECISIONS.md`](DECISIONS.md).

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — operation duration metrics and signals
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage, phases, and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
- [`../reference/product-specification.md`](../reference/product-specification.md) — sections 14.3, 17.3, and 19.1
