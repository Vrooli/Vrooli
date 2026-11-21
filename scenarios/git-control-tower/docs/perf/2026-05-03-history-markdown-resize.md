---
date: 2026-05-03
scenario: git-control-tower
interactions:
  - history-panel-mount-and-scroll
  - markdown-file-open-and-preview
  - sidebar-col-resize-drag
traces:
  before: /tmp/git-control-tower/perf/trace.json
  capture_script: /tmp/git-control-tower/perf/capture.js
status: open
related_skill_run: scenario-performance-audit
---

# Perf audit: History list, Markdown viewer, Panel resize

First perf audit of git-control-tower. Generic baseline triggered by a
"look at lists, markdown rendering, and resizing" request — no specific
user complaint anchored a single hot interaction. Phase 1 added the
canonical perf-build infra (vite profile mode, lib/profiler.ts, top-level
`<React.Profiler>`, and inner boundaries around `MarkdownPreview`,
`GitHistory`, `DiffViewer`).

## Framing

- User complaint (verbatim if possible): none — generic targeting of
  "lists (history panel), markdown rendering (file viewer), and
  resizing (resizeable panels)".
- Environment: local desktop browser, headless Chromium (rebrowser-
  playwright via BAS), 1440×900 viewport.
- Reproduction trigger: open the app pointing at the Vrooli repo (76
  visible file rows, multi-thousand-commit history). Phase A wheels the
  history panel up/down. Phase B clicks the first `.md` row
  (`docs/agent-system/INTAKE_PIPELINE.md`) then clicks the "Preview" tab.
  Phase C drags the first `cursor-col-resize` divider ±80 px three times.

## Methodology

- Profile-mode build verified: served `assets/index-CpU5ncH0.js` contained
  `onProfilerRender`, `MarkdownPreviewImpl`, `GitHistoryImpl`,
  `DiffViewerImpl`, `StateStackImpl`.
- Capture script: `/tmp/git-control-tower/perf/capture.js`
- Interactions exercised, in order: shell mount-settle (1.2 s) → history
  panel mouse-wheel ±400 px × 8 each direction → click first `.md` file
  row → click "Preview" toggle → drag col-resize handle ±80 px × 3
  cycles (75 mousemoves total).
- Capture configuration: viewport 1440×900, `tracingComplete` timeout
  120 s, recordMode `recordContinuously`, included categories include
  `blink.user_timing`, `disabled-by-default-v8.cpu_profiler`,
  `devtools.timeline`. Trace size 19.6 MB, 71,966 events, 432 ⚛ user-
  timing entries, 853 CPU `ProfileChunk` events.

## Per-component aggregation

| component        | count | total(ms) | avg(μs) | max(μs) |
| ---------------- | ----: | --------: | ------: | ------: |
| App              |    75 |    1102.6 |  14,702 |  60,001 |
| GitHistory       |    51 |     212.7 |   4,171 |   9,001 |
| DiffViewer       |    55 |      66.1 |   1,202 |  40,000 |
| MarkdownPreview  |     6 |      41.5 |   6,917 |  39,201 |

## Long-task summary

| metric    | before | after | delta |
| --------- | -----: | ----: | ----: |
| count     |      2 |   n/a |   n/a |
| total(ms) |    135 |   n/a |   n/a |
| max(ms)   |     73 |   n/a |   n/a |

(Discovery audit only; no after-trace yet.) FCP 128 ms, LCP 340 ms.
The two long tasks (73 ms, 62 ms) are both at first paint / markdown first
render; the resize phase produced **zero** long tasks > 50 ms despite
generating 75 App commits — each commit fits inside one frame today, but
only barely.

## Findings

### Finding 1 — Resize state at App scope re-renders the entire tree on every mousemove

- **What:** `App.tsx:87` (`sidebarWidth`), `App.tsx:92` (`changesHeight`),
  `App.tsx:97` (`historyHeight`), `App.tsx:1739–1840` (three resize
  `useEffect` blocks each calling `setX(...)` per mousemove).
- **Evidence:** App component shows 75 commits / 1102.6 ms total / avg
  14.7 ms / max 60.0 ms in the table above. The capture's resize phase
  generated exactly 25 mousemoves × 3 cycles = 75 events — a 1:1 map to
  App commits. Cascade: GitHistory's 51 commits inherit from App
  re-renders because its props are inline arrows from App.tsx:1973–2008.
- **Hypothesis:** App-level `useState` for resize widths/heights forces a
  full top-of-tree re-render every mousemove. App owns ~26 useState
  slots and ~25 query/mutation subscriptions, so its render is heavy and
  cascades through every consumer.
- **Suggested next step:** during drag, write width/height to a `ref` and
  update DOM imperatively (`element.style.width = ...`). Commit to React
  state only on `mouseup`. The `isResizing*` boolean flags can stay at
  App scope since they only flip twice per drag.

### Finding 2 — App-level state and query churn drives baseline commit cardinality

- **What:** `App.tsx:80–162` (~26 `useState` + ~25 subscription hooks);
  polling intervals 10–30 s on ~12 queries (`lib/hooks-core.ts:59–245`,
  `lib/hooks-agent.ts`, `lib/hooks-visual.ts`).
- **Evidence:** GitHistory still committed 51 times across the ~15 s
  window even outside the resize phase, far above the ~3-poll-tick
  expectation. Inline-arrow callbacks defeat any future memoization.
- **Hypothesis:** App is a god-component. Every state mutation fans out
  to the entire tree because consumers are direct children.
- **Suggested next step:** lift resize/layout state into a dedicated
  `<LayoutShell>` boundary; lift history-filter state into
  `<HistoryPanel>`; convert modal open-flags into per-modal disclosure
  hooks so unrelated modal state changes don't re-render siblings.

### Finding 3 — `GitHistory` renders all visible commits non-virtualized

- **What:** `components/GitHistory.tsx:541` (`{visibleEntries.map(...)}`,
  ~150-line per-entry render body).
- **Evidence:** avg 4.2 ms per commit at the current ~30-entry data
  shape. With `onLoadMore` paging up to thousands of commits this scales
  linearly into the 80–150 ms range per render.
- **Hypothesis:** The list will become a major hotspot once history grows
  beyond a few hundred entries.
- **Suggested next step:** virtualize `visibleEntries` with a windowed
  list. If a virtualization lib is added, also use it in `FileList`
  (Finding 5).

### Finding 4 — `MarkdownPreview` first render dominates a single frame

- **What:** `components/MarkdownPreview.tsx:249` (`MarkdownPreviewImpl`),
  `components/MarkdownPreview.tsx:252–375` (`components` map with ~25
  inline-arrow tag components).
- **Evidence:** 6 commits, avg 6.9 ms, **max 39.2 ms** for a ~30 KB
  markdown document. Surfaces as the 62 ms long-task in
  `trace.web-vitals.json`.
- **Hypothesis:** `react-markdown` parsing + React node construction for
  the entire document on the commit thread. Will be visible jank on
  larger docs (PROGRESS.md, generated reports, large READMEs). The
  shiki highlighter import (`MarkdownPreview.tsx:59`) is already lazy,
  so code blocks are *not* compounding the first render.
- **Suggested next step:** progressively reveal the doc by splitting on
  top-level headings and rendering one section per frame
  (`startTransition` + `useDeferredValue` of a section index). Or gate
  rendering on doc size with an explicit "render anyway" affordance.

### Finding 5 — Visible-but-unmeasured: `FileList` is a non-virtualized 76-row list

- **What:** `components/FileList.tsx`, `components/FileSection.tsx`,
  `components/FileRow.tsx` — 76 rows rendered with no windowing in the
  trace's working set.
- **Evidence:** No Profiler boundary was added around `FileList` in this
  audit, so cost is not in the table. DOM inspection during capture
  showed 76 `[data-file-path]` rows. By analogy with the BacklogTab
  pattern in swarm-manager, the cost is likely material once row counts
  exceed a few hundred.
- **Suggested next step:** add `<Profiler id="FileList">` boundary in a
  follow-up audit; virtualize with the same lib chosen for Finding 3 if
  per-commit cost exceeds ~5 ms or row counts grow.

## Recommendations + outcome

See `docs/perf/2026-05-03-after-fixes.md` for the comparison-run delta table
and per-recommendation evidence.

| # | Recommendation                                                           | Status | Notes                                                                    |
| - | ------------------------------------------------------------------------ | ------ | ------------------------------------------------------------------------ |
| 1 | DOM-imperative panel resize (lift width/height out of App `useState`)    | fixed | App avg/commit -28%. Resize-specific setStates eliminated; residual App count is polling-cascade driven. |
| 2 | Lift App layout/filter/modal state into dedicated subtree boundaries     | deferred | Gated: App count still 74 but driven by polling, not resize. Structural refactor scoped as a separate plan. |
| 3 | Virtualize `GitHistory` `visibleEntries`                                 | fixed | -64% avg/commit. `@tanstack/react-virtual` `^3.13.24` with `measureElement`. |
| 4 | Chunk `MarkdownPreview` rendering by top-level heading + `startTransition` | inconclusive | Implemented as `useDeferredValue`; total CPU unchanged, long-task delta flat-to-worse on this capture's interaction. Kept for input-yield property. |
| 5 | Add `<Profiler id="FileList">` and virtualize if hot                     | partial | Boundary added; FileList now confirmed hot (avg 17.7 ms × 35 commits). Virtualization deferred — multi-section structure needs flattened-list refactor; dedicated follow-up. |

## New dependencies

- `@tanstack/react-virtual` *or* `react-window` for Findings 3 + 5
  (author preference; not yet authorized).

## Notes

- Scenario has no `docs/manifest.json` yet, so this doc cannot be
  registered for the in-app docs viewer. Creating one is out of scope
  for this audit; surface it as a separate task if the in-app docs
  viewer matters here.
- `knowledge-observatory docs template perf-audit` and
  `knowledge-observatory docs audit` are not yet wired in this build of
  the CLI (the docschema package recognises the type, but no `audit` /
  `template` subcommands are exposed). This doc was authored by copying
  `scenarios/knowledge-observatory/api/internal/docschema/templates/PERF-AUDIT.md`
  manually. When those CLI commands land, re-validate by running them
  against this scenario.
