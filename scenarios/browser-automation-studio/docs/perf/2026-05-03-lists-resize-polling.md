---
date: 2026-05-03
scenario: browser-automation-studio
interactions:
  - dashboard projects/executions tab traversal and list scroll
  - executions tab polling settle window
  - record mode mount, sidebar tab switching, and sidebar resize drag
traces:
  before: /tmp/browser-automation-studio/perf/trace.json
  after: /tmp/browser-automation-studio/perf/trace.after.json
  follow_up_narrow_boundaries: /tmp/browser-automation-studio/perf/trace.narrow-memo.json
  follow_up_viewport_guard: /tmp/browser-automation-studio/perf/trace.viewport-guard.json
  capture_script: /tmp/browser-automation-studio/perf/capture.js
status: implemented
related_skill_run: scenario-performance-audit
---

# Perf audit: lists, resize, and polling

## Framing

- User complaint: "measure the performance of the browser-automation-studio scenario ... Look into anything where there are lists, resizing, or any sort of polled data shown"
- Environment: local desktop browser profile build, headless Chromium via Browser Automation Studio's existing `rebrowser-playwright` install.
- Reproduction trigger: dashboard list/tab traversal, executions polling settle window, record mode mount, sidebar tab clicks, and sidebar resize drags.
- Data limitation: local fixture state had zero project/workflow/execution rows, so list cardinality pressure was not measured. The trace is still useful for idle/polling and resize churn.

## Methodology

- Profile-mode build verified: `onProfilerRender`, `RecordModePage`, and profiler IDs were present in the built chunks under `ui/dist`.
- Capture script: `/tmp/browser-automation-studio/perf/capture.js`
- Trace: `/tmp/browser-automation-studio/perf/trace.json`
- Web vitals: `/tmp/browser-automation-studio/perf/trace.web-vitals.json`
- Capture configuration: 1440x900 viewport, Chrome tracing categories from `scenario-performance-audit`, ~15 second interaction window.

## Per-component aggregation

Before:

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---:|---:|---:|---:|
| App | 6816 | 2830.2 | 415 | 11500 |
| RecordModePage | 6766 | 2480.9 | 367 | 11500 |
| UnifiedSidebar | 121 | 31.7 | 262 | 1000 |
| DashboardContent | 15 | 29.5 | 1966 | 10099 |
| ExecutionsTab | 1 | 9.7 | 9700 | 9700 |

After:

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---:|---:|---:|---:|
| App | 6816 | 2861.0 | 420 | 11300 |
| RecordModePage | 6766 | 2491.7 | 368 | 11200 |
| DashboardContent | 15 | 32.9 | 2194 | 10101 |
| UnifiedSidebar | 120 | 29.3 | 244 | 1300 |
| ExecutionsTab | 1 | 9.6 | 9601 | 9601 |

Phase counts:

| component | mount | update | nested-update |
|---|---:|---:|---:|
| App | 1 | 6812 | 3 |
| RecordModePage | 1 | 6765 | 0 |
| UnifiedSidebar | 1 | 120 | 0 |
| DashboardContent | 1 | 14 | 0 |
| ExecutionsTab | 1 | 0 | 0 |

## Long-task summary

| metric | before | after | delta |
|---|---:|---:|---:|
| count | 2 | 2 | 0 |
| total(ms) | 192.0 | 230.1 | +38.1 |
| max(ms) | 106.0 | 132.4 | +26.4 |

Paint/LCP from the same run:

| metric | value |
|---|---:|
| first-paint | 132 ms |
| first-contentful-paint | 132 ms |
| largest-contentful-paint | 216 ms |

## Findings

1. **Record mode commits thousands of times during a short idle/resize window.**  
   Evidence: `RecordModePage` committed 6,766 times, accounting for 2,480.9 ms of React commit time. The likely source is frame-stream state and parent callbacks: [useFrameStream.ts](../../ui/src/domains/recording/capture/useFrameStream.ts:192) polls as fast as 300 ms, but successful frame handling also updates `isFetching`, `displayedTimestamp`, dimensions, frame stats, and stream status around [useFrameStream.ts](../../ui/src/domains/recording/capture/useFrameStream.ts:604), [useFrameStream.ts](../../ui/src/domains/recording/capture/useFrameStream.ts:688), [useFrameStream.ts](../../ui/src/domains/recording/capture/useFrameStream.ts:696), and [useFrameStream.ts](../../ui/src/domains/recording/capture/useFrameStream.ts:717). Those changes bubble into parent state through [RecordPreviewPanel.tsx](../../ui/src/domains/recording/timeline/RecordPreviewPanel.tsx:88) and [RecordingSession.tsx](../../ui/src/domains/recording/RecordingSession.tsx:1459).

2. **Sidebar resize itself is not the dominant React cost, but it writes React state on every mousemove.**  
   Evidence: `UnifiedSidebar` committed 121 times for only 31.7 ms total, max 1.0 ms. The implementation updates width on every document `mousemove` at [useUnifiedSidebar.ts](../../ui/src/domains/recording/sidebar/useUnifiedSidebar.ts:297) and persists storage both on mouseup and in a resize-end effect at [useUnifiedSidebar.ts](../../ui/src/domains/recording/sidebar/useUnifiedSidebar.ts:306) and [useUnifiedSidebar.ts](../../ui/src/domains/recording/sidebar/useUnifiedSidebar.ts:321). This is acceptable in the measured run, but it will scale with pointer event rate and any heavier descendants.

3. **Dashboard/executions polling is guarded, but duplicated intervals can diverge.**  
   Evidence: dashboard content committed only 15 times and `ExecutionsTab` mounted once in the empty-data run. Polling is gated by running-execution count in [DashboardView.tsx](../../ui/src/views/DashboardView/DashboardView.tsx:124), but `ExecutionsTab` also installs a separate 5 second interval at [ExecutionsTab.tsx](../../ui/src/domains/executions/history/ExecutionsTab.tsx:63). With running rows present, that duplicates refresh ownership and can produce extra list churn.

4. **Execution list filtering recomputes multiple passes per render and lacks virtualization.**  
   Evidence: the local trace had zero execution rows, so this is a code-risk finding rather than a measured row-count bottleneck. `ExecutionsTab` builds `allExecutions`, then filters for the active list and counts statuses with multiple full-array passes at [ExecutionsTab.tsx](../../ui/src/domains/executions/history/ExecutionsTab.tsx:120) and [ExecutionsTab.tsx](../../ui/src/domains/executions/history/ExecutionsTab.tsx:137). Rows are rendered with direct `.map` calls at [ExecutionsTab.tsx](../../ui/src/domains/executions/history/ExecutionsTab.tsx:209). This is fine for small histories but will become visible with large execution histories or frequent polling.

5. **Node palette renders every visible card and performs highlight splitting in render.**  
   Evidence: this route was not reached in the capture, so this is also a code-risk finding. `NodeCard` calls `highlightText` for label and description during render at [NodePalette.tsx](../../ui/src/domains/workflows/builder/NodePalette.tsx:164), while the palette maps every visible favorite/recent/category node at [NodePalette.tsx](../../ui/src/domains/workflows/builder/NodePalette.tsx:500) and [NodePalette.tsx](../../ui/src/domains/workflows/builder/NodePalette.tsx:551). Search updates will rerender all visible cards.

## Recommendations + outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Decouple live frame delivery from whole `RecordModePage` React state. Keep per-frame timestamp/fetching/dimensions in refs or a small external store selected by only the chrome/stats widgets. | partial | Frame timestamp, latency, fetching, and dimension propagation now avoid per-frame React state where possible. The broad `RecordModePage` profiler boundary still reports the same subtree commit count, so a narrower `PlaywrightView`/frame-stream boundary should be used next to separate canvas stream work from parent state work. |
| 2 | Throttle parent-visible frame stats/status to a display cadence, e.g. 500-1000 ms, and avoid propagating `displayedTimestamp` every frame unless the visible status text is enabled. | implemented | `useFrameStats` now batches at 1000 ms, `displayedTimestamp` state is throttled to 1000 ms, connection status callbacks are deduped, and latency stats stay in refs. |
| 3 | Make sidebar resize use `requestAnimationFrame` or CSS variable updates during drag, commit React width once on mouseup, and persist once. | implemented | Resize width updates are RAF-coalesced, final width is committed on mouseup, and storage persistence uses the final width. |
| 4 | Consolidate running-execution polling into one owner and use WebSocket/store updates for list changes. | implemented | Removed the duplicate `ExecutionsTab` interval; dashboard/store refresh remains the owner. |
| 5 | Memoize execution filtering/counting in one `useMemo`, memoize `ExecutionCard`, and add virtualization if histories can exceed roughly 100 rows. | implemented without virtualization | `ExecutionsTab` and global executions now use single-pass filtering/counting, and row cards are memoized. Virtualization remains deferred until seeded histories show enough row pressure to justify it. |
| 6 | For the node palette, memoize `NodeCard`, memoize highlighted fragments by node/search term, and consider virtualizing the category list if custom node packs grow large. | implemented without virtualization | `NodeCard` and highlighted fragments are memoized, search uses `useDeferredValue`, and palette callbacks are stable. Virtualization remains deferred pending large custom packs. |

## New dependencies

- None required for recommendations 1-4 and the memoization parts of 5-6.
- Optional: `@tanstack/react-virtual` if execution history or palette virtualization is implemented.

## Follow-up: narrower live-preview boundaries

Implemented after the initial recommendations:

- Added profiler boundaries for `RecordingSession`, `PreviewContainer`, `BrowserChrome`, `RecordPreviewPanel`, and `PlaywrightView`.
- Moved recording `pageTitle`, `frameStats`, and stream `connectionStatus` out of `RecordingSession` React state and into selected `sessionStore` slices.
- Let `RecordingHeader` and `BrowserChrome` subscribe directly to the live metadata they display.
- Disabled timestamp React state in `PlaywrightView` when the in-preview connection indicator is hidden.
- Memoized `BrowserChrome`, `RecordPreviewPanel`, and `PlaywrightView` so parent/subtree commits do not force stable chrome work.

Follow-up trace: `/tmp/browser-automation-studio/perf/trace.narrow-memo.json`

| component | previous after count | follow-up count | previous after total(ms) | follow-up total(ms) |
|---|---:|---:|---:|---:|
| App | 6816 | 9430 | 2861.0 | 2060.9 |
| RecordModePage | 6766 | 8702 | 2491.7 | 1532.8 |
| RecordingSession | n/a | 8648 | n/a | 1488.8 |
| PreviewContainer | n/a | 7369 | n/a | 988.1 |
| RecordPreviewPanel | n/a | 4341 | n/a | 471.3 |
| PlaywrightView | n/a | 987 | n/a | 99.3 |
| BrowserChrome | n/a | 116 | n/a | 39.3 |
| UnifiedSidebar | 120 | 113 | 29.3 | 30.6 |

Interpretation: commit counts vary with delivered frames, so total duration and narrow-boundary attribution are the more useful signals here. The broad page boundary still commits often because it encloses the live preview subtree, but total `RecordModePage` commit time dropped by about 959 ms, and `BrowserChrome` is no longer visited at frame cadence. The remaining hot path is below `PreviewContainer`/`RecordPreviewPanel`, with `PlaywrightView` itself only ~99 ms total in the run.

## Follow-up: viewport/context guard

Implemented after the narrow-boundary pass:

- Added a `StablePreviewWrapper` profiler boundary.
- Added viewport equality guards in `ViewportSyncManager.updateFromBounds` so reporting the same clamped bounds no longer updates viewport state or refreshes the provider value.
- Tightened `PreviewContainer`'s viewport-reporting effect so it only reports changed browser viewport dimensions.

Follow-up trace: `/tmp/browser-automation-studio/perf/trace.viewport-guard.json`

| component | narrow-memo count | viewport-guard count | narrow-memo total(ms) | viewport-guard total(ms) |
|---|---:|---:|---:|---:|
| App | 9430 | 183 | 2060.9 | 209.9 |
| RecordModePage | 8702 | 145 | 1532.8 | 149.5 |
| RecordingSession | 8648 | 145 | 1488.8 | 127.6 |
| PreviewContainer | 7369 | 134 | 988.1 | 59.4 |
| BrowserChrome | 116 | 120 | 39.3 | 41.9 |
| StablePreviewWrapper | n/a | 76 | n/a | 10.4 |
| RecordPreviewPanel | 4341 | 42 | 471.3 | 5.9 |
| PlaywrightView | 987 | 13 | 99.3 | 1.3 |

Interpretation: the main remaining preview churn was the viewport feedback loop, not canvas drawing. Guarding unchanged bounds cut `RecordModePage` from 8,702 commits to 145 and `PlaywrightView` from 987 commits to 13 in the same capture flow. The live canvas path is now largely isolated from replay presentation and viewport synchronization work when dimensions are stable.
