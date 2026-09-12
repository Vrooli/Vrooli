# React Coherence Notes

## Last Updated
2026-05-27

## Baseline surface UX contract (Plan C)

Every baseline-includable review tab (Screenshots, Workflows, Tests, Rules)
renders an identical baseline-awareness layer from one shared primitive set in
`ui/src/features/baselines/`. Do not re-implement these per tab:

- **`SurfaceCaptureEmptyState`** — the only "nothing captured yet" state. Two
  intents: capture-loose (the tab's own run trigger, no manifest) and
  capture-baseline (`SetBaselineModal` pre-scoped to the surface).
- **`SurfaceBaselineBar` + `BaselineSelector`** — the only "you have data" header.
  States loose-vs-baseline viewing, switches the default baseline, runs/exits
  compare, jumps to the Baselines tab.
- **`SurfaceComparePanel` + `useCompareOnDemand`** — the only compare path
  (Decision 3). `useCompareOnDemand` is the single compare-trigger hook; both it
  and `BaselineCompareView` consume it — never re-add a local
  `useState(started) + useBaselineDiff(enabled)` pair.
- **`useSurfaceBaselineModal`** — the only "Capture baseline" affordance wiring
  (modal open state + `preselectedSurfaces` + default-on-create).

"Baseline" in the UI means a `BaselineManifest` only — there is no UI action that
creates a `role=baseline` visual snapshot (Decision 1).

## State Management Patterns
[How state flows through the application]
- Global state approach:
- Component-local state patterns:
- Server state / cache strategy:

## Duplication Audit
[Identified duplication across components]
- **Area**: what's duplicated, where, recommended consolidation

## Styling Patterns
[CSS/styling approach and consistency]
- Approach: [CSS modules, styled-components, Tailwind, etc.]
- Theme tokens used:
- Inconsistencies found:

## Component Coherence
[Patterns that should be consistent across components]
- Prop naming conventions:
- Error handling patterns:
- Loading state patterns:

## Recommendations
1. [Highest priority coherence improvement]
2. [Second priority]
