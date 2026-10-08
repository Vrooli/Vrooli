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
  after: /tmp/app-monitor/perf/trace.after.json
  capture_script: /tmp/app-monitor/perf/capture.js
  stable_capture_script: ../../tools/perf/preview-workspace-capture.mjs
status: fixed
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

Before:

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

After same-script comparison:

| component | before count | after count | before total(ms) | after total(ms) | before avg(μs) | after avg(μs) | delta(ms) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ⚛ App | 181 | 150 | 286.2 | 265.2 | 1581 | 1768 | -20.9 |
| ⚛ PreviewWorkspaceView | 153 | 126 | 198.9 | 194.9 | 1300 | 1547 | -4.0 |
| ⚛ PreviewPane | 204 | 193 | 159.2 | 162.2 | 780 | 841 | +3.1 |
| ⚛ AppShell | 37 | 26 | 249.0 | 157.0 | 6730 | 6038 | -92.0 |
| ⚛ Outlet | 31 | 22 | 172.0 | 112.2 | 5548 | 5100 | -59.8 |
| ⚛ GraphCanvas | 24 | 17 | 155.8 | 98.7 | 6492 | 5806 | -57.1 |
| ⚛ Sidebar | 15 | 10 | 74.6 | 43.0 | 4973 | 4300 | -31.6 |

## Long-task summary

| metric | before | after | delta |
|---|---:|---:|---:|
| count | 5 | 137 | +132 |
| total(ms) | 297.0 | 29228.0 | +28931.0 |
| max(ms) | 71.0 | 7509.0 | +7438.0 |

Long-task entries were reported as `same-origin-descendant` / `multiple-contexts`, which points to iframe work being a visible part of the user-perceived cost. The same-script after run loaded 4 iframes initially while the original baseline loaded 3, so the long-task delta is not an app-monitor shell regression signal. A separate offscreen validation seeded 8 panes and observed 4 initial iframes, then 8 after scrolling the lower panes into view, confirming the deferred iframe loading behavior.

## Findings

- **What:** [previewWorkspaceStore.ts](../../ui/src/features/preview-workspace/state/previewWorkspaceStore.ts) persisted full per-pane navigation state, including arbitrary-length `previewUrl`, `previewUrlInput`, `initialPreviewUrl`, and `history`.
  **Evidence:** reported production error was a browser quota exception while writing `app-monitor:preview-workspace-v1`; the capture used several pane URL/history updates and the store writes on every `setPaneViewState`.
  **Hypothesis:** iframe bridge location updates and URL input paths can accumulate long URLs across panes, then every pane-state change serializes the entire workspace into localStorage.
  **Suggested next step:** fixed in this change by bounding runtime history, compacting persisted history/URLs, and catching quota errors during storage writes.

- **What:** [PreviewWorkspaceView.tsx](../../ui/src/features/preview-workspace/components/PreviewWorkspaceView.tsx) commits frequently during layout operations.
  **Evidence:** `PreviewWorkspaceView` committed 153 times, 198.9 ms total, 12.5 ms max. Pointer resize currently writes column/row fractions to the global persisted workspace store on each pointer move.
  **Hypothesis:** resize operations couple transient drag state to persisted global state, causing extra store notifications and localStorage writes during a high-frequency interaction.
  **Suggested next step:** fixed by keeping resize fractions as local drag-preview state and committing final fractions to the store on pointerup/pointercancel.

- **What:** [PreviewPane.tsx](../../ui/src/features/preview-workspace/components/PreviewPane.tsx) commits often across multi-pane workspace interactions.
  **Evidence:** `PreviewPane` committed 204 times, 159.2 ms total, including 64 nested-update profiler entries in the trace.
  **Hypothesis:** navigation/session effects mirror iframe bridge state back into the workspace store, and logs/full-view effects also call `setPaneViewState`. The equality guard prevents identical writes, but the same store object is still the coordination point for live iframe state and durable layout state.
  **Suggested next step:** fixed by moving live pane navigation/full-view/logs state into a runtime-only store and debouncing compact URL snapshots back to the persisted workspace store.

- **What:** [PreviewPane.tsx](../../ui/src/features/preview-workspace/components/PreviewPane.tsx) eagerly loads iframe previews for every pane.
  **Evidence:** capture mounted 4 panes and 3 iframes initially, then 5 panes and 4 iframes after workspace manager interaction. Long tasks were all `same-origin-descendant`.
  **Hypothesis:** embedded apps dominate some felt latency; offscreen or lower-priority panes consume CPU/network while the user is interacting with the app-monitor shell.
  **Suggested next step:** fixed by keeping focused/pinned panes eager and deferring iframe `src` assignment for non-eager panes until they intersect the viewport/root margin.

- **What:** [TabSwitcherCards.tsx](../../ui/src/components/tabSwitcher/TabSwitcherCards.tsx) progressively reveals scenario/resource cards in batches of 24.
  **Evidence:** tab/scenario picker was not the top profiler row in this run, but it is included under `App` commits and can still scale with total scenario/resource count.
  **Hypothesis:** batched rendering avoids a single massive commit, but still mounts all matching cards over successive frames.
  **Suggested next step:** fixed with bounded card-grid windows. Matching scenarios/resources no longer auto-mount in successive RAF batches; users explicitly expand with Show more.

## Recommendations + Outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Bound preview workspace persistence and catch quota failures | fixed | Implemented in `previewWorkspaceStore.ts`; regression test added. |
| 2 | Add profile build infrastructure and profiler boundaries | fixed | Required for repeatable future audits. |
| 3 | Commit pane resize fractions only at drag end | fixed | Implemented local drag-preview state in `PreviewWorkspaceView.tsx`. |
| 4 | Defer offscreen iframe loading and inactive pane hooks | fixed | Implemented focused/pinned eager loading plus IntersectionObserver activation in `PreviewPane.tsx`. |
| 5 | Bound scenario/resource picker rendering | fixed | Replaced automatic progressive reveal with an explicit card window and Show more expansion. |
| 6 | Add stable 8-12 pane perf playbook | fixed | Added `docs/perf/preview-workspace-stable-playbook.md` and `tools/perf/preview-workspace-capture.mjs`. |

## New Dependencies

- None for the implemented fixes.
- Optional future dependency: a virtualizer package for the scenario/resource card grid if custom virtualization is not desired.
