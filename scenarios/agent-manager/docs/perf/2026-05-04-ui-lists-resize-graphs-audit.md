---
date: 2026-05-04
scenario: agent-manager
interactions:
  - dashboard mount
  - runs list search, scroll, selection, and master-detail resize
  - run detail resize and task/timeline/diff/cost tab switching
  - tasks and profiles list search plus master-detail resize
  - stats charts, tables, and time-window changes
traces:
  before: /tmp/agent-manager/perf/trace.json
  after:
  capture_script: /tmp/agent-manager/perf/capture.js
status: open
related_skill_run: scenario-performance-audit
---

# Perf audit: UI lists, resize, and graphs

## Framing

- User complaint: "measure the performance of the agent-manager scenario ... suggest improvements ... all of the lists, resizeable stuff, graphs, etc."
- Environment: local lifecycle-managed `agent-manager`, profile production UI bundle served at `http://localhost:21238`, captured headlessly in Chromium at 1440x900.
- Reproduction trigger: route through dashboard, runs/tasks/profiles master-detail list surfaces, resize panel handles, switch run-detail tabs, then exercise stats charts and tables.

## Methodology

- Profile-mode build readiness was missing and was added before capture: Vite profile aliases, `build:profile`, `src/lib/profiler.ts`, top-level `App` profiler, page profilers, `MasterDetail:*` profilers, and stats-section profilers.
- Profile bundle verified by grepping the served asset for `onProfilerRender`, `RunsPage`, and `StatsPage`.
- Capture script: `/tmp/agent-manager/perf/capture.js`
- Trace artifacts: `/tmp/agent-manager/perf/trace.json` and `/tmp/agent-manager/perf/trace.web-vitals.json`
- Capture configuration: Chromium headless, viewport 1440x900, CDP categories included `blink.user_timing`, `v8.execute`, devtools timeline, screenshots, and CPU profiler chunks.
- Trace sanity: 2,356 React `⚛` user-timing entries, 3,634 CPU profile chunks, 230,911 trace events.

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---:|---:|---:|---:|
| ⚛ App | 347 | 2671.8 | 7700 | 130401 |
| ⚛ RunsPage | 126 | 1535.7 | 12188 | 126601 |
| ⚛ MasterDetail:runs | 119 | 1413.5 | 11878 | 114800 |
| ⚛ TasksPage | 64 | 776.1 | 12127 | 111601 |
| ⚛ MasterDetail:tasks | 64 | 740.2 | 11566 | 107901 |
| ⚛ StatsPage | 96 | 165.2 | 1721 | 18300 |
| ⚛ Stats:RunStatusTrends | 65 | 53.5 | 823 | 12100 |
| ⚛ Stats:ToolUsageAnalytics | 49 | 40.5 | 827 | 5100 |
| ⚛ ProfilesPage | 35 | 33.1 | 946 | 4600 |
| ⚛ MasterDetail:profiles | 35 | 29.4 | 840 | 4200 |
| ⚛ Stats:CostDurationTrends | 65 | 27.9 | 429 | 6201 |
| ⚛ Stats:ModelUsageBreakdown | 36 | 26.6 | 739 | 15600 |
| ⚛ DashboardPage | 12 | 15.2 | 1267 | 3200 |
| ⚛ Stats:KPISummary | 11 | 3.1 | 282 | 701 |
| ⚛ Stats:ErrorAnalysisSection | 9 | 2.4 | 267 | 1200 |
| ⚛ Stats:ProfileActivityTable | 9 | 2.2 | 245 | 1200 |
| ⚛ Stats:RunnerPerformanceTable | 8 | 1.7 | 213 | 700 |

## Long-task summary

| metric | before | after | delta |
|---|---:|---:|---:|
| count | 0 |  |  |
| total(ms) | 0 |  |  |
| max(ms) | 0 |  |  |

The long-task observer recorded zero browser long tasks for this scripted run. The React profile still shows large commit costs in the master-detail list surfaces; treat those as component rendering headroom rather than proven user-blocking stalls.

## DOM Observations

| phase | list/clickable nodes | SVG nodes | separators | table rows | notes |
|---|---:|---:|---:|---:|---|
| dashboard | 8 | 31 | 0 | 0 | lightweight route |
| runs initial | 652 | 2084 | 1 | 0 | list + selected detail surface mounted together |
| runs final | 788 | 1593 | 1 | 0 | detail tabs and resize exercised |
| tasks | 788 | 1663 | 1 | 0 | same master-detail pressure pattern as runs |
| profiles | 16 | 47 | 1 | 0 | comparatively small |
| stats | 1 | 12 | 0 | 0 | chart section cheap in this data set |

## Findings

- **What:** [RunsPage.tsx](../../ui/src/pages/RunsPage.tsx) renders every filtered run row and the full selected-run detail subtree inside one master-detail commit path.
  **Evidence:** `RunsPage` consumed 1,535.7 ms across 126 commits, 12,188 μs average, 126,601 μs max; `MasterDetail:runs` consumed 1,413.5 ms across 119 commits, 11,878 μs average. The runs phase mounted 652-788 clickable/list nodes and 1,593-2,084 SVG nodes.
  **Hypothesis:** search, selection, route-sync, tab switching, and resize all re-render the complete list plus detail area. The list maps all visible rows in [RunsPage.tsx](../../ui/src/pages/RunsPage.tsx) around line 530, and each row receives fresh inline callbacks/actions around lines 535, 559, and 647.
  **Suggested next step:** split `RunList` and `RunListRow` into memoized components with stable callbacks, precomputed `taskTitleById`/`profileNameById` maps, and row-level action components. If large run histories are common, virtualize the list so resize and detail-tab updates do not reconcile every row.

- **What:** [TasksPage.tsx](../../ui/src/pages/TasksPage.tsx) shows the same master-detail scaling risk as runs when the task list is large.
  **Evidence:** `TasksPage` consumed 776.1 ms across 64 commits, 12,127 μs average, 111,601 μs max; `MasterDetail:tasks` consumed 740.2 ms across 64 commits, 11,566 μs average. The scripted task phase logged 788 clickable/list nodes and 1,663 SVG nodes.
  **Hypothesis:** the list maps all filtered tasks in [TasksPage.tsx](../../ui/src/pages/TasksPage.tsx) around line 460, with search/sort recomputing around line 352 and resize updates causing the whole master-detail subtree to commit.
  **Suggested next step:** share a bounded/virtualized list primitive across runs/tasks/profiles. Start with memoized row extraction and stable callbacks; add virtualization only if real task counts exceed a few dozen.

- **What:** [useResizablePanel.ts](../../ui/src/hooks/useResizablePanel.ts) updates React state on every mousemove and persists every intermediate size to localStorage.
  **Evidence:** resize was part of the highest-cost phases: `MasterDetail:runs` max 114,800 μs and `MasterDetail:tasks` max 107,901 μs. The hook calls `setSize` in the raw `mousemove` handler in [useResizablePanel.ts](../../ui/src/hooks/useResizablePanel.ts) around line 130 and writes `localStorage` on every size change around line 83.
  **Hypothesis:** while dragging, every pointer event invalidates the full master-detail subtree and synchronously writes storage. This cost compounds with non-virtualized lists.
  **Suggested next step:** keep drag preview size in a ref or local CSS variable and commit React state plus localStorage only on mouseup, or throttle to one `requestAnimationFrame` update per frame and persist only the final size.

- **What:** top-level [App.tsx](../../ui/src/App.tsx) remains coupled to broad API/websocket state and propagates many updates into active route trees.
  **Evidence:** `App` consumed 2,671.8 ms across 347 commits, including 13 nested-update commits. It merges all runs with websocket snapshots in [App.tsx](../../ui/src/App.tsx) around line 61 and hydrates status updates from websocket messages around line 78.
  **Hypothesis:** every run snapshot map change produces a new `mergedRuns` array and causes route-level re-rendering. On runs/tasks surfaces this fans out into the expensive master-detail list trees.
  **Suggested next step:** move websocket-derived run projection closer to consumers with selectors keyed by active route/run IDs. For the dashboard, keep the existing limited projection; for runs, separate list row summaries from selected-run detail state.

- **What:** stats graphs are not the main bottleneck in this capture, but chart sections still commit frequently on time-window changes.
  **Evidence:** `StatsPage` consumed 165.2 ms total; the heaviest stats children were `Stats:RunStatusTrends` at 53.5 ms and `Stats:ToolUsageAnalytics` at 40.5 ms. No long tasks were recorded during stats interactions.
  **Hypothesis:** Recharts rendering is acceptable for the current data volume, but chart data arrays are rebuilt on each render in [RunStatusTrends.tsx](../../ui/src/features/stats/components/trends/RunStatusTrends.tsx) around line 41 and [ToolUsageAnalytics.tsx](../../ui/src/features/stats/components/breakdown/ToolUsageAnalytics.tsx) around line 36.
  **Suggested next step:** keep stats as lower priority. If the stats endpoint grows, memoize chart data transforms by API response identity and cap bar/table detail lists before considering chart-library changes.

## Recommendations + Outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Keep the profile-build infrastructure and Profiler boundaries | fixed | Added as an audit prerequisite; needed for future repeatable measurements. |
| 2 | Decouple resize drag preview from persisted React state | open | Highest leverage because it affects runs/tasks/profiles master-detail layouts. |
| 3 | Memoize and/or virtualize runs/tasks list rows | open | Runs is the top hotspot; tasks shows the same shape. |
| 4 | Split route-wide run state into list summaries and selected-run detail state | open | Reduces `App`/`RunsPage` fan-out from websocket and detail updates. |
| 5 | Defer stats graph optimization | deferred | Current graphs were measurable but not dominant; optimize only if larger stats payloads reproduce lag. |

## New Dependencies

- Optional: `@tanstack/react-virtual` if the team wants a proven virtualizer instead of a small in-house bounded list/windowing primitive.
