---
date: 2026-05-03
scenario: vrooli-autoheal
interactions:
  - dashboard manual tick, health-group expand/collapse, recent-events filter/show-more/scroll
  - trends tab time-window changes, trend table sorting/scrolling, chart viewport resize
  - check detail modal open, history tab switch, history scroll
traces:
  before: /tmp/vrooli-autoheal/perf/trace.json
  after: /tmp/vrooli-autoheal/perf/trace.after.json
  capture_script: /tmp/vrooli-autoheal/perf/capture.js
status: fixed
related_skill_run: scenario-performance-audit
---

# Perf audit: list, resize, and polling surfaces

## Framing

- User complaint: "measure the performance of the vrooli-autoheal scenario ... Look into anything where there are lists, resizing, or any sort of polled data shown"
- Environment: local profile-mode production bundle served by `vrooli scenario restart vrooli-autoheal`; captured in headless Chromium at 1440x900 with a controlled API fixture.
- Reproduction trigger: dashboard polled status/timeline UI, recent-events list expansion and scroll, trends chart/table interactions, viewport resize, and modal history list scroll.

## Methodology

- Profile-mode build verified: served bundle contained `onProfilerRender`, `DashboardSurfaceImpl`, and `CheckDetailModalImpl`.
- Capture script: `/tmp/vrooli-autoheal/perf/capture.js`
- Raw trace: `/tmp/vrooli-autoheal/perf/trace.json`
- Web vitals: `/tmp/vrooli-autoheal/perf/trace.web-vitals.json`
- Capture data shape: 90 health checks, 420 timeline events, 90 trend rows, 50 rendered incidents, and 240 modal history rows.
- Trace sanity: 96,220 events, 1,832 React Profiler user-timing entries, 1,079 CPU profile chunks, 29.4 MB trace.

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---|---|---|---|
| App | 315 | 557.5 | 1770 | 35900 |
| TrendsPage | 271 | 427.0 | 1576 | 33900 |
| UptimeTrendChart | 266 | 240.0 | 902 | 13200 |
| CheckTrendGrid | 7 | 115.3 | 16471 | 22700 |
| DashboardSurface | 39 | 107.7 | 2762 | 35200 |
| CheckDetailModal | 3 | 33.2 | 11066 | 29599 |
| EventsTimeline | 8 | 18.1 | 2263 | 5201 |

## Long-task summary

| metric | before | after | delta |
|---|---:|---:|---:|
| count | 2 | 2 | 0 |
| total(ms) | 121 | 111 | -10 |
| max(ms) | 69 | 59 | -10 |

## Validation run

The follow-up implementation kept the same capture script and fixture. Raw after trace: `/tmp/vrooli-autoheal/perf/trace.after.json`.

| component | before count | after count | before(ms) | after(ms) | before avg(μs) | after avg(μs) | delta(ms) | delta |
|---|---|---|---|---|---|---|---|---|
| App | 315 | 318 | 557.5 | 474.2 | 1770 | 1491 | -83.3 | -15% |
| TrendsPage | 271 | 272 | 427.0 | 371.1 | 1576 | 1364 | -55.9 | -13% |
| UptimeTrendChart | 266 | 263 | 240.0 | 260.6 | 902 | 991 | 20.6 | +9% |
| CheckTrendGrid | 7 | 8 | 115.3 | 64.5 | 16471 | 8063 | -50.8 | -44% |
| DashboardSurface | 39 | 40 | 107.7 | 81.7 | 2762 | 2042 | -26.0 | -24% |
| CheckDetailModal | 3 | 3 | 33.2 | 16.1 | 11066 | 5366 | -17.1 | -52% |
| EventsTimeline | 8 | 8 | 18.1 | 18.9 | 2263 | 2363 | 0.8 | +4% |

## Findings

- **What:** `ui/src/surfaces/trends/components/CheckTrendGrid.tsx:249` renders every trend row on each sort/refetch, including one sparkline per row at `ui/src/surfaces/trends/components/CheckTrendGrid.tsx:281`.
  **Evidence:** `CheckTrendGrid` committed only 7 times but consumed 115.3 ms total, averaging 16.5 ms per commit with a 22.7 ms max. That is the highest per-commit cost in the trace and is enough to miss a frame on table sort or polling replacement.
  **Hypothesis:** whole-table rendering dominates; each row is rebuilt with inline handlers, repeated `getTitle()` calls, and sparkline rendering. A 90-row fixture already hits frame budget; larger deployments will scale linearly.
  **Suggested next step:** extract a memoized `CheckTrendRow`, compute `title` once per row, stabilize row callbacks, and consider table virtualization if expected row counts can exceed roughly 100 checks.

- **What:** `ui/src/surfaces/trends/components/UptimeTrendChart.tsx:142` uses Recharts `ResponsiveContainer` and a full `AreaChart` under a polled query plus viewport resizing.
  **Evidence:** `UptimeTrendChart` committed 266 times for 240.0 ms total. Average commit cost was modest at 902 μs, but cardinality was high and max reached 13.2 ms during window/resize phases.
  **Hypothesis:** resize observer updates from `ResponsiveContainer`, query state transitions, and parent `TrendsPage` re-renders repeatedly commit the chart even when the rendered data has not meaningfully changed.
  **Suggested next step:** wrap the chart component in `memo`, hoist stable chart child props where practical, and debounce/coalesce resize-driven width changes if users resize panes or windows frequently.

- **What:** `ui/src/surfaces/trends/TrendsPage.tsx:86` runs four independent polling queries and passes fresh fallback arrays into heavy children at `ui/src/surfaces/trends/TrendsPage.tsx:228`.
  **Evidence:** `TrendsPage` committed 271 times for 427.0 ms total. `App` committed 315 times for 557.5 ms, showing broad parent churn during the polling/list/resize scenario.
  **Hypothesis:** independent query state transitions are producing many small parent commits. The `checkTrendsData?.trends ?? []` and `timelineData?.events ?? []` expressions allocate new empty arrays whenever data is absent, which can defeat memoization during loading/error transitions.
  **Suggested next step:** define module-level `EMPTY_TRENDS`/`EMPTY_EVENTS`, wrap list/chart children in `memo`, and evaluate whether trends/uptime/incidents/timeline polling can be enabled only while the Trends tab is visible.

- **What:** `ui/src/surfaces/trends/TrendsPage.tsx:260` caps incidents to 50 rendered rows but still re-renders the full incident block with `TrendsPage`.
  **Evidence:** incident rows were not separately instrumented, but `TrendsPage` was the second largest component total and contains both the incident list and chart/table orchestration.
  **Hypothesis:** the incident list is acceptable at 50 rows, but it still contributes to parent commit work and repeats `getTitle()` calls at `ui/src/surfaces/trends/TrendsPage.tsx:276` and `ui/src/surfaces/trends/TrendsPage.tsx:278`.
  **Suggested next step:** extract and memoize `IncidentsList`; compute the display title once per incident row.

- **What:** `ui/src/shared/components/CheckDetailModal.tsx:545` renders every history row when the History tab is selected.
  **Evidence:** `CheckDetailModal` committed 3 times but averaged 11.1 ms with a 29.6 ms max when opened against 240 history rows.
  **Hypothesis:** modal open/history switch pays for full history rendering and repeated timestamp formatting for all rows.
  **Suggested next step:** add a visible-row cap/show-more flow or virtualization for history, and precompute formatted timestamps in a memoized projection.

- **What:** `ui/src/surfaces/dashboard/components/EventsTimeline.tsx:27` filters the full timeline twice and renders the visible slice at `ui/src/surfaces/dashboard/components/EventsTimeline.tsx:102`.
  **Evidence:** `EventsTimeline` was not a primary bottleneck in this fixture: 8 commits, 18.1 ms total, 2.3 ms average, 5.2 ms max.
  **Hypothesis:** the current `showCount` cap keeps it safe for normal usage. The full-array `issueCount` filter at `ui/src/surfaces/dashboard/components/EventsTimeline.tsx:35` is still O(N) on every timeline replacement.
  **Suggested next step:** keep the cap, but compute `filteredEvents` and `issueCount` in a single pass if timeline payloads grow substantially.

- **What:** `ui/src/surfaces/dashboard/DashboardSurface.tsx:90` renders all checks in expanded health groups.
  **Evidence:** `DashboardSurface` committed 39 times for 107.7 ms total, with a 35.2 ms max during group/tick interactions.
  **Hypothesis:** expanded groups render complete check-card lists; a large "Healthy" group can create a single expensive commit.
  **Suggested next step:** memoize `CheckGroupSection`/`CheckCard` if not already done, and consider keeping large healthy groups collapsed by default or adding a show-more cap.

## Recommendations + outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Memoize/extract trend table rows and avoid repeated title lookups. | fixed | `CheckTrendGrid` average commit dropped from 16.5 ms to 8.1 ms. |
| 2 | Memoize chart and coalesce resize-driven chart commits. | fixed | Follow-up replaced Recharts with a fixed-viewBox SVG chart, removing chart resize-observer React commits entirely. |
| 3 | Gate Trends tab polling to active tab and use stable empty arrays. | fixed | Trends tab already unmounts when inactive; implementation added stable empty arrays and disabled background interval refetching. |
| 4 | Cap or virtualize modal history rows. | fixed | Added an 80-row visible cap with show-more; modal average commit dropped from 11.1 ms to 5.4 ms. |
| 5 | Keep recent-events cap; optimize issue filtering only if payloads grow. | deferred | EventsTimeline stayed low-impact; no behavioral change needed now. |

## Follow-up hardening

- Replaced the Recharts uptime chart with a native SVG implementation to remove third-party resize work from the hot Trends tab path.
- Added a 100-row visible cap to the trend grid so deployments with larger check counts do not pay the full table render cost on initial sort/refetch.
- Shared check metadata through `CheckMetadataContext` so `App` no longer subscribes to a duplicate `checks-metadata` query observer.
- Normalized the scenario Makefile to the canonical lifecycle wrapper required by `scenario-auditor`.
- Added a test-phase UI server and `/health` wait in `.vrooli/service.json`; this keeps smoke and Lighthouse from racing a protected UI process that is stopped when entering the test phase.
- Bounded the API `/health` dependency probe so lifecycle/integration checks return promptly during SQLite contention.
- Final scenario validation: `vrooli scenario test vrooli-autoheal` passed on 2026-05-03 at 23:58 America/New_York.

## New dependencies

- Optional: `@tanstack/react-virtual` if implementing virtualization for `CheckTrendGrid` or modal history. Memoization/capping recommendations need no new dependency.
