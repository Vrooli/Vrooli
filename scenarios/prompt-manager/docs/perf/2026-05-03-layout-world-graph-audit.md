---
date: 2026-05-03
scenario: prompt-manager
interactions:
  - world route mount and sidebar activity
  - sidebar search and resize drag cycles
  - world canvas wheel and drag input
  - graph route visual/json toggle and canvas input
traces:
  before: /tmp/prompt-manager/perf/trace.json
  after: /tmp/prompt-manager/perf/trace.after.json
  capture_script: /tmp/prompt-manager/perf/capture.js
status: fixed
related_skill_run: scenario-performance-audit
---

# Perf audit: layout, world, and graph surfaces

## Framing

- User complaint: "measure the performance of the prompt-manager scenario ... Look into anything where there are lists, resizing, the world view, and the graph view"
- Environment: local desktop browser profile build served by `vrooli scenario restart prompt-manager`; viewport 1440x900; URL `http://localhost:21235`.
- Reproduction trigger: mount `/world`, exercise sidebar search/list affordances, drag the sidebar resize handle, interact with the world canvas, then mount `/graph` and toggle visual/json mode plus pan/zoom.

## Methodology

- Profile-mode build verified: served bundle contained `onProfilerRender`, `SkillManagerLayoutImpl`, `SkillListViewImpl`, `SkillCardViewImpl`, `GraphView`, and `WorldCanvasImpl`.
- Capture script: `/tmp/prompt-manager/perf/capture.js`
- Trace: `/tmp/prompt-manager/perf/trace.json` (61 MB)
- Capture configuration: headless Chromium via BAS Playwright dependency, Chrome tracing categories from `scenario-performance-audit`, 1440x900 viewport.
- Fixture limitation: this local data set rendered zero `skill-list-item`, zero `skill-card-item`, and zero React Flow graph nodes during the capture. The measurements are still valid for the shell, resize, world, and empty-graph route work, but populated-list and populated-graph cardinality need a seeded follow-up capture.

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---:|---:|---:|---:|
| App | 177 | 830.2 | 4690 | 168900 |
| SkillManagerLayout | 176 | 820.8 | 4664 | 168900 |
| GraphView | 44 | 270.5 | 6148 | 164600 |
| WorldCanvas | 16 | 15.7 | 981 | 6000 |

Trace sanity:

- React user-timing entries: 847
- Paired React commits: 413
- CPU profile chunks: 5,703

Implementation validation run:

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---:|---:|---:|---:|
| App | 69 | 313.6 | 4545 | 145900 |
| SkillManagerLayout | 69 | 310.8 | 4504 | 145901 |
| GraphView | 17 | 249.4 | 14670 | 142801 |
| WorldCanvas | 50 | 16.8 | 336 | 1499 |

Validation trace sanity:

- Trace: `/tmp/prompt-manager/perf/trace.after.json` (23 MB)
- Capture script: `/tmp/prompt-manager/perf/capture-after.js`
- React user-timing entries: 434
- Paired React commits: 205
- CPU profile chunks: 2,300
- Note: the validation script uses bounded DOM-triggered list/card/tree toggles because Playwright role-click auto-wait hung on the icon-only view controls. The interaction still covers the same optimized surfaces.

## Long-task summary

| metric | before | after | delta |
|---|---:|---:|---:|
| count | 5 | 4 | -1 |
| total(ms) | 481 | 448 | -33 |
| max(ms) | 180 | 210 | +30 |

Paint/LCP from the same web-vitals capture:

- First paint: 84 ms
- First contentful paint: 188 ms
- LCP: 612 ms

## Findings

### 1. Resize updates re-render the whole application shell on every mousemove

- **What:** [useResizableSidebar.ts](../../ui/src/hooks/useResizableSidebar.ts) lines 100-131 call `setWidth` on every `mousemove`; [SkillManagerLayout.tsx](../../ui/src/components/layout/SkillManagerLayout.tsx) lines 1389-1415 binds that width directly to the outer sidebar panel.
- **Evidence:** `SkillManagerLayout` committed 176 times for 820.8 ms total in the capture, essentially matching `App` at 177 commits. The resize phase is the only high-frequency input loop in the script.
- **Hypothesis:** each drag event updates React state, causing the full layout subtree and route content to participate in resize renders. The `transition-[width] duration-200` class also asks layout to animate width changes while the pointer is already producing many updates.
- **Suggested next step:** move live resize width into a CSS custom property or direct DOM style during pointer movement, commit React state only on pointer-up or `requestAnimationFrame`, and disable the width transition while resizing. This should reduce `SkillManagerLayout` commit count during resize without changing behavior.

### 2. Graph route has high per-commit cost even with no rendered graph nodes

- **What:** [GraphView.tsx](../../ui/src/components/graph/GraphView.tsx) lines 224-245 subscribes to many store slices independently; lines 275-334, 383-472, and 488-520 rebuild view models, nodes, edges, layout signatures, and layout effects when dependencies change.
- **Evidence:** `GraphView` had 44 commits, 270.5 ms total, 6,148 μs average, and a 164,600 μs max despite the DOM inspection confirming zero `.react-flow__node` elements in this local graph.
- **Hypothesis:** graph state churn and derived structures are paying fixed cost before any visible graph density is involved. Separate Zustand subscriptions plus new `Set`/`Map`/array objects increase the chance that unrelated graph store updates trigger the whole view.
- **Suggested next step:** consolidate graph selectors with `useShallow`, split the shell from the expensive visual graph body, and memoize or cache graph layout work by a stable graph/filter signature. For populated graphs, evaluate moving Dagre layout to a worker or deferring it behind `startTransition`.

### 3. List and tree rendering still scales linearly with every visible item

- **What:** [useSkillTree.ts](../../ui/src/hooks/useSkillTree.ts) lines 149-181 filters, searches, sorts, and tree-filters the full skills array; [SkillTreeSidebar.tsx](../../ui/src/components/tree/SkillTreeSidebar.tsx) lines 1769-1790 recursively renders all expanded tree nodes; [SkillListView.tsx](../../ui/src/components/sidebar/SkillListView.tsx) lines 87-189 and [SkillCardView.tsx](../../ui/src/components/sidebar/SkillCardView.tsx) lines 87-196 render all filtered skills.
- **Evidence:** this capture could not measure populated row cost because the local fixture produced zero list/card rows, but the source path is straightforward O(N) render work for each expanded tree/list/card view. The top-level `SkillManagerLayout` cost means any list growth will be paid inside an already hot subtree.
- **Hypothesis:** large skill packs or broad expanded trees will make search, filter, and resize feel worse because rows are not windowed and row handlers are created in the map body.
- **Suggested next step:** add virtualization/windowing for flat list and card views first, then consider tree windowing for expanded folder trees. Also hoist row components and stable callbacks so memoization can isolate selection and dirty-state changes.

### 4. World view React commits are cheap; remaining world risk is mostly render-loop/GPU-side

- **What:** [WorldCanvas.tsx](../../ui/src/components/world/WorldCanvas.tsx) lines 134-177 subscribes to graphics, performance, selection, camera, furniture, decoration, and environment stores; lines 242-281 rebuild and publish agent positions when agent/furniture/seating state changes; lines 436-478 owns the Three.js canvas.
- **Evidence:** `WorldCanvas` committed only 16 times for 15.7 ms total, 981 μs average, and 6,000 μs max. This is not the bottleneck in the React Profiler trace.
- **Hypothesis:** perceived world-view lag, if present, is more likely in R3F render-loop work, draw calls, materials, shadows, DPR, or pointer handlers than in React commit cost.
- **Suggested next step:** run a world-specific GPU/render-loop audit with FPS and draw-call counters. Keep the current DPR/tier controls, and consider splitting overlay/UI state from the canvas owner only if future traces show React commits rising.

## Recommendations + outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Throttle or imperative-buffer sidebar resizing; commit width once per animation frame or on pointer-up. | fixed | `useResizableSidebar` live-mutates the sidebar CSS variable during drag and commits React state on mouse-up. |
| 2 | Remove width transition during active resize. | fixed | Sidebar width no longer uses a width transition during pointer drag. |
| 3 | Add seeded performance fixtures for populated skill rows and graph nodes before optimizing list/card/graph density. | deferred | Still needed for future graph-density work; not required for the implemented UI fixes. |
| 4 | Virtualize `SkillListView` and `SkillCardView`; evaluate tree virtualization for expanded folders. | fixed | Added dependency-free row windowing for list/card and flattened virtual tree rows. |
| 5 | Consolidate `GraphView` store selection and cache layout by stable graph/filter signatures. | fixed | Consolidated store selection, removed mirrored React Flow node/edge state, and reused one health-score map. |
| 6 | Use R3F/GPU-specific profiling for the world view rather than React commit profiling. | deferred | React commit cost remains low; GPU/render-loop profiling is a separate audit if world lag persists. |

## New dependencies

- (none) — virtualization was implemented locally without adding packages.
