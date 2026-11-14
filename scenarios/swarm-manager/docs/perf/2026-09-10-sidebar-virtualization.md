---
date: 2026-09-10
scenario: swarm-manager
interactions:
  - sidebar-tab-switch
  - sidebar-idle-polling
  - sidebar-search-typing
  - backlog-tab-scroll
traces:
  before: /tmp/claude-1000/-home-matthalloran8-Vrooli/3454482f-f3ce-4c49-a38b-1aaee4682c65/scratchpad/baseline.json
  after: /tmp/claude-1000/-home-matthalloran8-Vrooli/3454482f-f3ce-4c49-a38b-1aaee4682c65/scratchpad/after.json
  capture_script: /tmp/claude-1000/-home-matthalloran8-Vrooli/3454482f-f3ce-4c49-a38b-1aaee4682c65/scratchpad/sidebar-perf.cjs
status: fixed
related_skill_run: performance
---

# Perf audit: Sidebar virtualization and idle re-renders

The operator reported the sidebar as slow. The cause was that every sidebar
list rendered every row: virtualization was on but never bounded, so the
Backlog tab mounted all 944 items (35,654 DOM nodes) and every background poll
re-rendered them.

## Methodology

`performance-health` could not start (its CLI fails to build on a missing
`go.sum` entry in the shared proto package; filed as
`knw-1789065070268697325`), and browser-automation-studio was unhealthy. The
capture was therefore a hand-rolled headless Chrome probe (playwright-core with
the system Chrome) against the running production build at `:21234`, 1440×900.
It records, per tab, rows mounted vs items, click-to-second-frame switch time
and long tasks; then long tasks over 20 s idle on the Backlog and Sessions
tabs, while typing "scenario" into search, and while scrolling the Backlog list.
Long tasks are the `longtask` PerformanceObserver entries (>50 ms). Runs vary
5–15%; every change below is far outside that band.

Data size at capture: 944 backlog items, 141 sessions, 126 scenarios,
33 executions, 32 goals, 20 captures.

## Findings

1. **The virtual list never got a height, so it rendered everything.**
   `CollectionList` passes `height="100%"` to `VirtualList`'s viewport, but the
   three wrappers between the sidebar's sized content box and that viewport
   (the list section, the collection container and the VirtualList section)
   were all auto-height. A percentage of an auto height is auto, so the
   viewport grew to the full content (291,055 px), every row counted as
   visible, and the outer sidebar div did the scrolling. The same pattern
   affects prompt-manager's `TopicListPanel`.
2. **Every poll replaced unchanged arrays.** `refreshActivities` (every 5 s)
   and `fetchSessions` (every 4 s while a session is active) always stored a
   new array, re-rendering the backlog list (through
   `useCommandPostItemActions`) and the tab badges, and rewriting the sessions
   cache in localStorage.
3. **Memoization was defeated.** Inline callbacks from `AppShell`
   (`onQuickCapture`) and `Sidebar` (`onCreateFromPlan`, the create action)
   gave the memoized tabs new props on each render, so each keystroke in the
   search box re-rendered the Backlog list before the debounced query landed.
   `BacklogCard` is memoized but received inline closures from `BacklogRow`.
4. **The next-actions query was keyed on the filtered view**, so each debounced
   search refetched next actions for the matching items.
5. **Tab badges regrouped the whole backlog on each session poll.**

## Fixes

- `CollectionList` 1.3.4 (react-component-library): a virtualized list marks its
  section `data-rcl-virtualized` and fills its container, with the wrappers as
  `flex: 1 1 0%` columns, so the viewport's height resolves. In an auto-height
  container the chain still sizes to its content, so no consumer loses rows.
  It also passes the memoized rows to `VirtualList` instead of a copy.
- `agent-activities-store` and `agent-session-store` keep the held array when a
  poll returns identical data (the same serialized-compare rule as
  `upsertSessionIfChanged`), and skip the localStorage write.
- Stable callbacks in `AppShell` and `Sidebar`; `BacklogRow` is memoized and
  passes stable handlers to `BacklogCard`.
- The next-actions query is keyed on the whole backlog.
- `SidebarTabs` groups action items only when backlog, captures, executions or
  snoozes change.

## Results

| Measure | Before | After |
|---|---|---|
| Backlog rows mounted / items | 944 / 944 | 7 / 944 |
| Sidebar DOM nodes, Backlog tab | 35,654 | 367 |
| Backlog tab switch (click → 2nd frame) | 291 ms, 1,950 ms long tasks (max 814) | 25 ms, none |
| Sessions tab switch | 194 ms, 1,083 ms long tasks | 32 ms, none |
| Captures tab switch (unmounting Backlog) | 590 ms, 1,105 ms long tasks | 47 ms, none |
| Idle 20 s, Backlog tab | 3,225 ms long tasks (17) | 566 ms (7) |
| Idle 20 s, Sessions tab | 2,089 ms long tasks (13) | 455 ms (7) |
| Typing 8 characters in search | 1,890 ms long tasks, slowest input 104 ms | none, slowest input 40 ms |
| Backlog scroll, frame p95 | 19 ms (outer div scrolls) | 17 ms (list viewport scrolls) |

The remaining idle long tasks are not the sidebar: with the sidebar collapsed
the same page shows 692 ms over 20 s against 623 ms with it open. They come
from the rest of the page and are not addressed here.

## Validation

- swarm-manager: `tsc --noEmit` clean; eslint clean on changed files; 114/114
  sidebar, store and shell tests pass.
- react-component-library: new `tests/components/CollectionList/1.3.4` suite
  passes; the only CollectionList failures are the 3 known ones in the 1.0.0
  suite, which test 1.3.x features against 1.0.0.
- jsdom does no layout, so the 1.3.4 test pins the markup and CSS contract; the
  browser behavior was verified by this probe, first with the rules injected
  into the running page and then on the rebuilt UI.
