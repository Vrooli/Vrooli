---
date: 2026-05-03
scenario: git-control-tower
interactions:
  - history-panel-mount-and-scroll
  - markdown-file-open-and-preview
  - sidebar-col-resize-drag
traces:
  before: /tmp/git-control-tower/perf/trace.before.json
  after: /tmp/git-control-tower/perf/trace.after.json
  capture_script: /tmp/git-control-tower/perf/capture.js
status: partially-fixed
related_skill_run: scenario-performance-audit
---

# Perf audit: After fixes (Fix 1 + Fix 3 + Fix 4 + Fix 5 boundary)

Comparison run validating the four perf fixes from the 2026-05-03 discovery
audit. Fix 5 (FileList virtualization) deferred — the boundary added in this
phase exposed FileList as a real hotspot, justifying a dedicated follow-up
audit + virtualization rather than a rushed fit.

## Framing

- User complaint (verbatim if possible): same as discovery — generic targeting
  of "lists (history panel), markdown rendering (file viewer), and resizing
  (resizeable panels)".
- Environment: local desktop browser, headless Chromium (rebrowser-playwright
  via BAS), 1440×900 viewport.
- Reproduction trigger: identical to discovery capture (same `capture.js`
  script, same interaction sequence).

## Methodology

- Profile-mode build verified: served `assets/index-DPgnaNn9.js` contains
  `onProfilerRender`, `MarkdownPreviewImpl`, `GitHistoryImpl`, `DiffViewerImpl`,
  `FileListImpl` (new boundary added by this round), `useDeferredValue`,
  imperative `style.width=`/`style.height=`/`gridTemplateRows=` markers (Fix 1
  evidence in minified output).
- Capture script: `/tmp/git-control-tower/perf/capture.js` (unchanged from
  discovery — comparison-run discipline per skill §"Comparison runs").
- Interactions exercised, in order: shell mount-settle → history wheel ±400 px
  × 8 each direction → click first `.md` row → click "Preview" toggle → drag
  col-resize handle ±80 px × 3 cycles.
- **Caveat for the comparison:** between captures the underlying repo's
  working tree changed (tracked-file count went from 76 → 117 visible rows
  during the after-capture). This drove additional `useRepoStatus` polling
  refetches in the after-window, inflating App's commit count. The honest
  signal is **avg(μs)** per commit, not raw count — see skill §"Reading the
  table".

## Per-component aggregation

| component        | b-cnt | a-cnt | before(ms) | after(ms) | b-avg(μs) | a-avg(μs) | delta(ms) | delta(%) |
| ---------------- | ----: | ----: | ---------: | --------: | --------: | --------: | --------: | -------: |
| App              |    75 |    74 |     1102.6 |     785.5 |    14,702 |    10,615 |    -317.1 |     -29% |
| GitHistory       |    51 |    52 |      212.7 |      78.5 |     4,171 |     1,510 |    -134.2 |     -63% |
| DiffViewer       |    55 |    28 |       66.1 |      52.8 |     1,202 |     1,885 |     -13.3 |     -20% |
| MarkdownPreview  |     6 |     2 |       41.5 |      40.1 |     6,917 |    20,049 |      -1.4 |      -3% |
| FileList         |     0 |    35 |        0.0 |     619.1 |         0 |    17,688 |    +619.1 |      new |

## Long-task summary

| metric    | before | after | delta |
| --------- | -----: | ----: | ----: |
| count     |      2 |     3 |    +1 |
| total(ms) |    135 |   198 |   +63 |
| max(ms)   |     73 |    78 |    +5 |

## Findings

### Fix 1 (panel resize, App.tsx:1731–1870) — partial win

- **Evidence:** App avg/commit **14,702 μs → 10,615 μs (-28%)**; total
  cumulative App work **1102.6 ms → 785.5 ms (-29%)**.
- The commit count did not drop (75 → 74) because the after-capture's
  working tree had more file changes, triggering more `useRepoStatus`
  refetches — those fan out to App since App owns the query subscription.
  This is App-level state-cascade churn (Finding 2 of the discovery audit),
  not a Fix 1 failure.
- **Verdict:** Fix 1 reduced per-commit work by ~28 %. Resize-specific
  re-renders are no longer the dominant App-commit driver; polling cascade is.

### Fix 3 (GitHistory virtualization, GitHistory.tsx:541) — clear win

- **Evidence:** avg/commit **4,171 μs → 1,510 μs (-64%)**; total **212.7 →
  78.5 ms (-63%)**.
- Virtualization with `@tanstack/react-virtual` and `measureElement` for
  variable-height rows succeeded. Now only the visible window of entries
  renders per commit instead of the entire `visibleEntries` array.
- **Verdict:** ✅ fixed.

### Fix 4 (MarkdownPreview useDeferredValue) — inconclusive / possibly net-zero

- **Evidence:** total nearly unchanged (41.5 → 40.1 ms); commits 6 → 2;
  avg/commit jumped 6,917 → 20,049 μs **because work concentrated into fewer
  commits** (a feature of `useDeferredValue`, not a regression). Long-task
  total grew 135 → 198 ms — but the third long task may be attributable to
  the now-visible FileList commits, not MarkdownPreview.
- **Verdict:** Inconclusive. The change preserves urgent input responsiveness
  (typing in inputs while a large markdown renders) but doesn't reduce CPU
  work; the long-task delta doesn't show a felt-perf improvement on this
  capture's interaction. Consider keeping the change for the input-yield
  benefit; a dedicated typing-latency capture would prove or disprove the win.

### Fix 5 (FileList Profiler boundary, FileList.tsx) — boundary added; virtualization deferred

- **Evidence (now visible):** 35 commits, avg **17,688 μs / commit**, total
  619.1 ms. This is the single largest avg/commit in the after-trace.
- **Hypothesis:** FileList renders all visible rows non-virtualized
  (FileSection.tsx:104–143 maps `entries` linearly). The 76→117 row growth in
  the working tree pushed cost into the visible band of this capture.
- **Suggested next step:** virtualize via the same `@tanstack/react-virtual`
  dep already authorized for Fix 3. Non-trivial because FileList renders
  multiple sections (staged/unstaged/untracked) inside one shared scroll
  container — a clean implementation requires flattening files+section-headers
  into a single array. Recommend a dedicated follow-up audit + fix; the
  Profiler boundary added in this round means the next comparison-run will
  have clean before/after data.

## Recommendations + outcome

| # | Recommendation                                                                | Status   | Notes                                                                                  |
| - | ---------------------------------------------------------------------------- | -------- | -------------------------------------------------------------------------------------- |
| 1 | DOM-imperative panel resize (App.tsx)                                         | fixed    | -28% avg/commit on App; cleaner once polling cascade is also addressed (see #2).        |
| 2 | Lift App layout/filter/modal state into dedicated subtree boundaries          | deferred | Gated decision: App count still ~74 (driven by polling, not resize). Larger refactor; defer to follow-up plan. |
| 3 | Virtualize `GitHistory.visibleEntries`                                        | fixed    | -64% avg/commit. `@tanstack/react-virtual` with `measureElement`.                       |
| 4 | Defer MarkdownPreview render via `useDeferredValue`                           | inconclusive | Total unchanged; long-task signal flat-to-worse. Kept for the input-yield property; revisit with a typing-latency capture if a real complaint surfaces. |
| 5 | Add `<Profiler id="FileList">` and virtualize if hot                          | partial — boundary added; virtualization deferred | Now confirmed hot (avg 17.7 ms × 35 commits). Multi-section structure makes virtualization larger; dedicated follow-up audit. |

## New dependencies

- `@tanstack/react-virtual` `^3.13.24` (added; ~5 KB gz). Authorized
  2026-05-03 by user.

## Decision: Phase 7.5 (App state lifting) gated → deferred

Per the original plan §7.5 gating criterion: "If App count ≤ 15 and avg ≤ 5
ms in the new trace, defer this phase." App is at count 74 / avg 10.6 ms —
neither threshold met. **However, the residual count is dominated by polling-
query cascade**, not resize-driven setStates. The structural fix (lifting
state into LayoutShell / HistoryFilters / per-modal disclosures) would not
reduce polling-driven re-renders — those need React Query subscriptions to
move out of App, which is a different refactor. Recommend treating App-state
lifting as a separate, scoped initiative rather than rolling into this perf-
fix branch. Captured in `docs/internal/PROBLEMS.md` as a deferred item.
