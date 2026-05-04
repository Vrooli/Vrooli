---
date: 2026-05-04
scenario: app-monitor
interactions:
  - preview workspace mount with four seeded panes
  - pane grid column resize
  - workspace scroll and minimap jump
  - workspace manager pane/control updates
  - tab switcher scenario picker search and segment changes
  - pane toolbar URL suggestions and scenario selector entry
traces:
  before: /tmp/app-monitor/perf/trace.before.json
  after: null
  capture_script: /tmp/app-monitor/perf/capture.js
status: open
related_skill_run: scenario-performance-audit
---

# Perf Audit: Preview Workspace

## Framing

- User complaint: "measure the performance of the app-monitor scenario ... Look into all of the iframe rendering, panes, scenario picker dialog" and investigate `QuotaExceededError` for `app-monitor:preview-workspace-v1`.
- Environment: local lifecycle-managed app-monitor profile build served at `http://localhost:20000`, captured headlessly with Chromium at 1440x900.
- Reproduction trigger: multi-pane preview workspace with iframe panes, grid resizing, minimap scrolling, workspace manager changes, tab/scenario picker search, and pane toolbar URL suggestions.

## Methodology

- Profile-mode build verified: served bundle contains `onProfilerRender`; added profile build support plus `App`, `PreviewWorkspaceView`, and `PreviewPane` profiler boundaries.
- Capture script: `/tmp/app-monitor/perf/capture.js`
- Interactions exercised, in order: seed four panes, wait for iframe previews, drag column splitter twice, scroll workspace/minimap, open workspace manager and mutate controls, open scenario picker and search, switch scenario/resource segments, exercise URL suggestion/scenario selector path.
- Capture configuration: Chromium headless, viewport 1440x900, CDP tracing categories include `blink.user_timing`, `v8.execute`, devtools timeline, screenshots, and CPU profiler chunks.

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---:|---:|---:|---:|
| ⚛ App | 181 | 286.2 | 1581 | 26800 |
| ⚛ AppShell | 37 | 249.0 | 6730 | 42200 |
| ⚛ PreviewWorkspaceView | 153 | 198.9 | 1300 | 12500 |
| ⚛ Outlet | 31 | 172.0 | 5548 | 42200 |
| ⚛ PreviewPane | 204 | 159.2 | 780 | 5700 |
| ⚛ GraphCanvas | 24 | 155.8 | 6492 | 41400 |
| ⚛ Sidebar | 15 | 74.6 | 4973 | 16601 |

Note: the trace includes same-origin iframe React profiler marks. `PreviewWorkspaceView` and `PreviewPane` are app-monitor boundaries; `AppShell`, `Outlet`, `GraphCanvas`, and `Sidebar` are from embedded scenario iframe content.

## Long-task summary

| metric | before | after | delta |
|---|---:|---:|---:|
| count | 5 |  |  |
| total(ms) | 297.0 |  |  |
| max(ms) | 71.0 |  |  |

Long-task entries were reported as `same-origin-descendant`, which points to iframe work being a visible part of the user-perceived cost.

## Findings

- **What:** [previewWorkspaceStore.ts](../../ui/src/features/preview-workspace/state/previewWorkspaceStore.ts) persisted full per-pane navigation state, including arbitrary-length `previewUrl`, `previewUrlInput`, `initialPreviewUrl`, and `history`.
  **Evidence:** reported production error was a browser quota exception while writing `app-monitor:preview-workspace-v1`; the capture used several pane URL/history updates and the store writes on every `setPaneViewState`.
  **Hypothesis:** iframe bridge location updates and URL input paths can accumulate long URLs across panes, then every pane-state change serializes the entire workspace into localStorage.
  **Suggested next step:** fixed in this change by bounding runtime history, compacting persisted history/URLs, and catching quota errors during storage writes.

- **What:** [PreviewWorkspaceView.tsx](../../ui/src/features/preview-workspace/components/PreviewWorkspaceView.tsx) commits frequently during layout operations.
  **Evidence:** `PreviewWorkspaceView` committed 153 times, 198.9 ms total, 12.5 ms max. Pointer resize currently writes column/row fractions to the global persisted workspace store on each pointer move.
  **Hypothesis:** resize operations couple transient drag state to persisted global state, causing extra store notifications and localStorage writes during a high-frequency interaction.
  **Suggested next step:** keep resize fractions as local drag-preview state and commit the final fractions to the store on pointerup/pointercancel.

- **What:** [PreviewPane.tsx](../../ui/src/features/preview-workspace/components/PreviewPane.tsx) commits often across multi-pane workspace interactions.
  **Evidence:** `PreviewPane` committed 204 times, 159.2 ms total, including 64 nested-update profiler entries in the trace.
  **Hypothesis:** navigation/session effects mirror iframe bridge state back into the workspace store, and logs/full-view effects also call `setPaneViewState`. The equality guard prevents identical writes, but the same store object is still the coordination point for live iframe state and durable layout state.
  **Suggested next step:** split durable workspace layout from volatile pane navigation runtime, then persist only compact durable snapshots on a debounce or browser idle callback.

- **What:** [PreviewPane.tsx](../../ui/src/features/preview-workspace/components/PreviewPane.tsx) eagerly loads iframe previews for every pane.
  **Evidence:** capture mounted 4 panes and 3 iframes initially, then 5 panes and 4 iframes after workspace manager interaction. Long tasks were all `same-origin-descendant`.
  **Hypothesis:** embedded apps dominate some felt latency; offscreen or lower-priority panes consume CPU/network while the user is interacting with the app-monitor shell.
  **Suggested next step:** add pane visibility detection and defer iframe `src` assignment for offscreen panes, or pause bridge/log/report hooks until a pane is focused or visible. Keep pinned/focused panes eager.

- **What:** [TabSwitcherCards.tsx](../../ui/src/components/tabSwitcher/TabSwitcherCards.tsx) progressively reveals scenario/resource cards in batches of 24.
  **Evidence:** tab/scenario picker was not the top profiler row in this run, but it is included under `App` commits and can still scale with total scenario/resource count.
  **Hypothesis:** batched rendering avoids a single massive commit, but still mounts all matching cards over successive frames.
  **Suggested next step:** if large catalogs become common, virtualize the card grid or hard-limit search results until the user refines the query.

## Recommendations + Outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Bound preview workspace persistence and catch quota failures | fixed | Implemented in `previewWorkspaceStore.ts`; regression test added. |
| 2 | Add profile build infrastructure and profiler boundaries | fixed | Required for repeatable future audits. |
| 3 | Commit pane resize fractions only at drag end | open | Should reduce store churn and localStorage writes during resize. |
| 4 | Defer offscreen iframe loading and inactive pane hooks | open | Highest expected iframe-related win; validate with same capture script. |
| 5 | Consider scenario/resource grid virtualization | deferred | No new dependency authorized; revisit if catalogs grow enough to make picker search expensive. |

## New Dependencies

- None for the implemented fixes.
- Optional future dependency: a virtualizer package for the scenario/resource card grid if custom virtualization is not desired.
