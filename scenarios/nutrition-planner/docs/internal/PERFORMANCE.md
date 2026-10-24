# Performance — Nutrition Planner

This document records performance budgets, current measurements, known
constraints, and regression procedures.

The working product name is **Nooch** (formerly Daily; decision D-042). Scenario id is `nutrition-planner`.

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
| Local control acknowledgement (redesign) | Ordinary UI actions acknowledge locally within **~100 ms** (a checkbox, a chip, a timer control, an equipment tile) (R25.4) | UI interaction capture on the reference hardware | declared, not measured |
| Cached navigation (redesign) | Switching between already-loaded destinations feels immediate; no full skeleton on a warm cache (R25.4) | UI navigation capture | declared, not measured |
| Common server writes (redesign) | Sub-second response absent external services (R25.4) | API timing in operation logs | declared, not measured |
| Hero media (redesign) | ~**250–600 KB** per optimized Today/recipe hero rendition where quality permits (R25.4) | Transfer size of the served rendition | declared, not measured |
| Collection thumbnails (redesign) | ~**30–100 KB** each (R25.4) | Transfer size per thumbnail | declared, not measured |
| Equipment scene (redesign) | ~**1 MB** of composite/layer bytes initially visible on compact screens (R25.4) | Transfer size at 390×844 | declared, not measured |
| Layout stability (redesign) | No layout shift when media loads, fails, or falls back: width/height or aspect ratio reserved for every image; titles and controls never jump (R07, R17.1) | Visual capture + cumulative layout shift from Lighthouse | declared, not measured |
| Asset loading discipline (redesign) | Load only the current appearance and composition; lazy-load below-fold images; never send full-resolution originals or both theme packs on first paint (R25.4) | Network capture per surface | declared, not measured |
| Lighthouse (scaffold config) | `.vrooli/lighthouse.json` gates one page (`/`, desktop) at performance ≥0.75 error / ≥0.85 warn and accessibility ≥0.90 / ≥0.95 | Test Genie performance phase | inherited; must be extended to the five destinations and a phone viewport when the redesign surfaces exist |

The media and interaction numbers are **review triggers, not reasons to
destroy image quality** (R25.4). An exception is allowed only with a measured
justification recorded in the ledger. They are engineering targets, never
presented to users as achieved guarantees.

The search budget is a **budget, not a promise**: a slow or timed-out
search returns its best validated draft plus a search status. A timeout
means "no feasible plan found in this search," never proof that none exists
(`PLN-06`, `ACT-047`). The numbers `20` candidates per slot, `8` branching,
beam width `~24`, and the ~2s budget are recommended configurable defaults
to be benchmarked, not hardcoded promises (`PLN-03`).

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-09-22 |
| No product workflow can be measured: every workspace RPC returns `401 unauthenticated` in the local runtime (blocker B1) and the live database holds no rows. | n/a | Audit in [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3 | 2026-09-22 |
| The current planner is a greedy picker with cost fixed at 0; its speed says nothing about the plan-level search the budget above describes. | n/a | `api/handlers/planning/connect_handler.go` | 2026-09-22 |
| No food or scene media exists yet, so media budgets have no measurement. | n/a | `ui/public` | 2026-09-22 |

The only currently measurable performance is the generated scaffold's
build and health behavior. Do not record scaffold timings as product
performance. Redesign measurements (interaction timings, media transfer
sizes, layout-shift results) are recorded per surface in
[`REDESIGN_LEDGER.md`](REDESIGN_LEDGER.md) with the viewport, appearance,
environment, and dataset, and summarized here once a surface is done.

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
- Media is the largest redesign cost. Generate responsive renditions once
  during asset processing (WebP or AVIF with a compatible fallback), keep
  originals out of the client, and record each rendition's bytes in the asset
  manifest (R17.4, R19.3).
- The equipment scene composes several layers; count their combined bytes at
  the compact viewport, not only the base image (R20, R25.4).
- Offline timers and the outbox must not poll: timers render from absolute
  timestamps, and the outbox replays on reconnect events (R13.3, R22).
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
5. For media budgets, capture the network log per surface at 390×844 and
   1440×1000 in both appearances and compare served bytes with the asset
   manifest; record the numbers in [`REDESIGN_LEDGER.md`](REDESIGN_LEDGER.md).
6. Record a persistent finding here if it is an accepted constraint, or in
   [`PROBLEMS.md`](PROBLEMS.md) if it is unresolved debt. A budget change is
   a durable decision and belongs in [`DECISIONS.md`](DECISIONS.md).

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — operation duration metrics and signals
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage, phases, and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
- [`REDESIGN_LEDGER.md`](REDESIGN_LEDGER.md) — per-surface redesign measurements
- [`../reference/product-specification.md`](../reference/product-specification.md) — R06.3, R07, R25.4, and Appendix A sections 14.3, 17.3, and 19.1
