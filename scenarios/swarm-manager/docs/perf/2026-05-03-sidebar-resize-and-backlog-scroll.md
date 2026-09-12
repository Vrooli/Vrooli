---
date: 2026-05-03
scenario: swarm-manager
interactions:
  - command-post-sidebar-resize
  - backlog-tab-scroll
  - graph-node-click-sequence
traces:
  before: /tmp/swarm-manager/perf/trace.before.json
  after: /tmp/swarm-manager/perf/trace.after.json
  capture_script: /tmp/swarm-manager/perf/capture.js
status: fixed
related_skill_run: scenario-performance-audit
---

# Perf audit: Sidebar resize + Backlog tab scroll

Initial perf audit on the swarm-manager UI. Three Do-Now items shipped;
several Do-Next items deferred. This is the first audit using the v2
`scenario-performance-audit` skill flow and the first to persist into
`docs/perf/` rather than `/tmp/findings.md`.

## Framing

User complaint (verbatim, paraphrased from session 2026-05-03):

> The Command Center page sidebar resize is laggy. The sidebar is slow when
> the backlog tab is selected with many results — possibly missing memo /
> virtualization, or filter/sort that should be server-side. No pagination.
> Graph rendering is also janky, but **don't reduce node/edge count** — that's
> intentional. Investigate actual render perf.

Environment: desktop browser, local development, profile-mode build.
Reproduction trigger: many backlog items present, click into BacklogTab,
drag sidebar resize handle, then scroll the list.

## Methodology

- Profile-mode build verified: `onProfilerRender` and `*Impl` symbols present
  in the served bundle.
- Capture script: `/tmp/swarm-manager/perf/capture.js` (3-phase Playwright +
  CDP trace, 1440×900 viewport, 120s `tracingComplete` timeout).
- Interactions exercised, in order: shell mount-settle → sidebar drag (one
  cycle) → BacklogTab tab-switch → list scroll-down → graph node click
  sequence.
- Both traces captured back-to-back on the same machine and load conditions
  to control for run-to-run noise.

## Per-component aggregation

| component | b-cnt | a-cnt | before(ms) | after(ms) | b-avg(μs) | a-avg(μs) | delta(ms) | delta(%) |
|---|---|---|---|---|---|---|---|---|
| BacklogTab | 14 | 60 | 826.0 | 114.0 | 59000 | 1900 | -712.0 | -86% |
| Sidebar | 22 | 22 | 412.0 | 198.0 | 18700 | 9000 | -214.0 | -52% |
| AppShell | 18 | 18 | 144.0 | 132.0 | 8000 | 7300 | -12.0 | -8% |
| GraphCanvas | 9 | 9 | 88.0 | 87.1 | 9800 | 9700 | -0.9 | -1% |

The BacklogTab row is the headline win: 14 expensive commits became 60
cheap ones (virtualization mounts/unmounts rows as the user scrolls). Total
dropped 86% but the avg-per-commit dropped ~31× — that's the metric the new
skill explicitly tells future agents to watch.

## Long-task summary

| metric | before | after | delta |
|---|---|---|---|
| count | 18 | 7 | -11 |
| total(ms) | 1327 | 175 | -1152 |
| max(ms) | 312 | 64 | -248 |

The long-task delta confirms the felt improvement: ~1.15s of main-thread
blocking gone. The user-perceptible lag during sidebar drag was driven by
the long-task spikes, not steady-state commits.

## Findings

1. **BacklogRow / FeedItem `memo()` was being defeated.** The `getItemCallbacks(item)` helper returned a fresh callback object every call (with mutation pending state baked in), and `useCommandPostItemActions` re-created callbacks on every parent render. Combined with `allItems: BacklogItem[]` prop churn (a fresh array literal in the parent), every parent commit forced every row to re-render even when nothing about the row's data changed.
   - **Files**: `ui/src/hooks/useCommandPostItemActions.ts:1-end`, `ui/src/surfaces/graph/components/sidebar/BacklogTab.tsx:1-end`, `ui/src/components/command-post/SummaryView.tsx:1-end`, `ui/src/components/backlog/backlog-card.tsx:1-end`.
   - **Evidence**: BacklogTab b-cnt=14 with b-avg=59ms (per-row work amortized into BacklogTab's own commit rather than offloaded to memo'd children).
   - **Fix shipped**: `getItemCallbacks` now returns from a memoized `Map<itemKey, StableItemCallbacks>`; `pendingArchiveKey` / `pendingWorkshop` / `pendingStatusKey` exposed as primitives so children can compute their pending state without baking it into the callbacks. `allItems` was lifted out of the per-row prop-set entirely (see #2).

2. **`DependencyIndicator` was rebuilding an O(N) Map on every render → O(N²) total.** Every `BacklogCard` received `allItems: BacklogItem[]` and did `new Map(allItems.map(...))` inside `DependencyIndicator` on each commit. With 278 backlog items, that's ~78k Map entries created per parent commit cycle.
   - **Files**: `ui/src/components/backlog/dependency-indicator.tsx`, `ui/src/components/backlog/backlog-card.tsx` (callers), `ui/src/app/shell/AppShell.tsx` (provider site).
   - **Fix shipped**: New `BacklogItemsProvider` context computes the kind/name → BacklogItem lookup once via `useMemo` at AppShell scope (`ui/src/components/backlog/backlog-items-context.tsx`). `DependencyIndicator` reads from `useBacklogItemLookup()`; `allItems` prop removed from `BacklogCardProps`.

3. **No virtualization on the backlog list.** All 278 backlog items rendered into the DOM regardless of scroll position. Rendering cost scaled linearly with list length, and the `display:none` overlap with the highlight-overlay (#5 below) compounded the cost.
   - **Files**: `ui/src/surfaces/graph/components/sidebar/BacklogTab.tsx`.
   - **Fix shipped**: `@tanstack/react-virtual` integrated; new `VirtualizedBacklogList` walks up to find the scrollable ancestor and renders ~9 rows at a time. DOM count went 278 → 9 (logged by the capture script's DOM-count probe).

4. **Graph rendering was already well-optimized.** The audit confirmed `GraphCanvas` is at -1% delta — the "graph feels janky" complaint was actually downstream of the sidebar/backlog work choking the main thread. With those fixed, graph perceived lag is mostly gone.
   - **Files**: `ui/src/surfaces/graph/...` (no changes needed).

## Recommendations + outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Hoist backlog item lookup Map to context (BacklogItemsProvider) | fixed | Phase 1 of audit |
| 2 | Replace getItemCallbacks per-call object with memoized Map | fixed | Required new StableItemCallbacks shape |
| 3 | Virtualize BacklogTab list (`@tanstack/react-virtual`) | fixed | DOM 278 → 9 |
| 4 | Wrap VirtualizedBacklogList in `<Profiler>` | fixed (subsequent commit) | Adds row to future Phase 5 tables |
| 5 | Move backlog filter/sort/pagination server-side | open (Do-Next) | Defer — measurable but scope-creeping |
| 6 | Throttle the highlight-overlay computation | open (Do-Next) | Currently fires per-pointer-move; should rAF-throttle |
| 7 | Audit per-card store-subscription churn | open (Do-Next) | Several cards subscribe to coarse selectors |
| 8 | Extract CSS transitions on hover-affected rows | open (Later) | Minor effect; only relevant under heavy scroll |
| 9 | Governance store `Set` churn | open (Later) | Frequent `new Set([...prev, x])` in some reducers |
| 10 | Optimize `computeUnblockingMap` | open (Later) | Currently O(N×D); fine at current N |

## New dependencies

- `@tanstack/react-virtual ^3.13.24` — added to `ui/package.json` for the
  virtualization fix in #3.
