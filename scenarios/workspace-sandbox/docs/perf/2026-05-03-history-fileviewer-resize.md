---
date: 2026-05-03
scenario: workspace-sandbox
interactions:
  - history-tab-load
  - select-history-archive
  - select-active-sandbox
  - sidebar-resize-3-cycles
  - header-resize-3-cycles
  - history-tab-second-visit
traces:
  before: /tmp/workspace-sandbox/perf/trace.json
  after: /tmp/workspace-sandbox/perf/trace.json
  capture_script: /tmp/workspace-sandbox/perf/capture.js
status: open
related_skill_run: scenario-performance-audit
---

# Perf audit: history list, file viewer, resizable panels

## Framing

- User complaint (verbatim): "measure the performance of the workspace-sandbox scenario … look into anything where there are lists (history panel), markdown rendering (file viewer), and resizing (resizable panels)"
- Environment: Headless Chromium 1440×900, profile-mode build served from `localhost:21239`, scenario co-resident on the development host (no tunnel).
- Reproduction trigger: Open the app cold, click through the History tab once, select the first archive, return to Active, drag the column-resize handle three cycles, drag the row-resize handle three cycles, click History again. Total interaction window ~12 s.
- Dataset shape at capture time: 9 active sandboxes, 8 history archives. The two largest archives have ≤2 files. The active sandbox set has zero file diffs. **No archive in the dataset exercises the markdown/source-view path** — `useHighlighting` only fires when `viewMode !== "diff"`, and `ClosedSandboxDetail` does not pass `viewMode`. The markdown-rendering findings below are derived statically and corroborated by long-task accounting; they were not measurable from this trace.

## Methodology

- Profile-mode build verified: `curl -s http://localhost:21239/$(MAIN) | grep -E 'onProfilerRender|[A-Z][a-zA-Z]+Impl'` returns `DiffViewerImpl`, `HistoryTabImpl`, `SandboxDetailImpl`, `onProfilerRender` — confirmed perf bundle.
- Capture script: `/tmp/workspace-sandbox/perf/capture.js`
- Interactions exercised, in order:
  1. Wait for `[data-testid='workspace-sandbox-app']`, settle 1.2 s
  2. Click `sidebar-tab-history`, settle 1.5 s
  3. Click first `sandbox-item` in history, settle 1.5 s
  4. Click `sidebar-tab-active`, settle 0.8 s
  5. Click first `sandbox-item` in active, settle 1.0 s
  6. Three forward+back drags of `sidebar-resize-handle` (±100 px, 30 substeps each)
  7. Three down+up drags of `header-resize-handle` (±80 px, 25 substeps each); the handle was present (selected sandbox shows the diff section even when empty), so phase F ran.
  8. Click `sidebar-tab-history` again, settle 0.8 s
- Capture configuration: viewport 1440×900, headless Chromium via `rebrowser-playwright`, CDP `Tracing.start` with the standard category set, 180 s timeout. Trace size 29 MB, 96 420 events, 2 128 React user-timing entries, 1 293 CPU profile chunks.

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---|---|---|---|
| App | 369 | 723.9 | 1962 | 5801 |
| SandboxDetail | 360 | 299.1 | 831 | 2000 |
| DiffViewer | 300 | 44.9 | 150 | 500 |
| HistoryTab | 5 | 8.7 | 1741 | 2800 |

Per-phase split (mount vs update):

| component+phase | count | total(ms) | avg(μs) | max(μs) |
|---|---|---|---|---|
| App / update | 369 | 723.9 | 1962 | 5801 |
| SandboxDetail / update | 359 | 297.1 | 828 | 1601 |
| DiffViewer / update | 298 | 44.5 | 149 | 500 |
| HistoryTab / update | 3 | 6.3 | 2100 | 2800 |
| HistoryTab / mount | 2 | 2.4 | 1201 | 2201 |
| SandboxDetail / mount | 1 | 2.0 | 2000 | 2000 |
| DiffViewer / mount | 2 | 0.4 | 201 | 300 |

## Long-task summary

| metric | value |
|---|---|
| count | 0 |
| total(ms) | 0 |
| max(ms) | 0 |

No browser long-tasks (>50 ms single ticks) registered. The bottleneck is **commit cardinality, not single-tick cost** — many cheap re-renders, none individually long enough to block the main thread for a full frame at this dataset size. With a larger active sandbox list, larger source files, or a slower device, several of these patterns will cross the long-task threshold; see findings below.

## Findings

### F1 — Sidebar drag re-renders the entire `App` tree on every mousemove

- **What:** `App.tsx:502` calls `setSidebarWidth(clampedWidth)` inside the `mousemove` handler installed on the window. The width is then applied via inline `style={{ width: sidebarWidth }}` on `App.tsx:917`. Every mousemove → `App` setState → full subtree re-render.
- **Evidence:** App had 369 commits in a 12 s window; ~330 of those line up with the resize phases (6 drags × {30 sidebar steps + 25 header steps}). Avg App commit cost 1962 µs, max 5801 µs. SandboxDetail tracks at 360 commits — the cascade reaches every right-pane subtree.
- **Hypothesis:** Width is hoisted to App-level state because the layout uses inline `style.width`. Each setState invalidates the entire memoization graph below App. The same pattern repeats for header height in `SandboxDetail.tsx:266` (`setHeaderHeight`).
- **Suggested next step:** Drive the panel width via a CSS custom property written directly to the container with `el.style.setProperty('--sidebar-w', px)` from a `requestAnimationFrame`-throttled mousemove handler, then commit the final value to React state on `mouseup`. The visual update bypasses React entirely; only the persisted "settled" width touches state. Same pattern for `headerHeight`. This is the highest-leverage fix in this audit.

### F2 — Resize handlers do synchronous `localStorage.setItem` on every step

- **What:** `App.tsx:526–528` and `SandboxDetail.tsx:290–293` register `useEffect`s that write `wsb.sidebarWidth` / `wsb.detailsHeight` to localStorage every time the value changes. During a drag, that's one synchronous DOM-storage write per mousemove (~30/s).
- **Evidence:** Not directly visible in the React Profiler track (effects fire after commit), but compounds the per-mousemove work surfaced in F1. localStorage writes are synchronous and serialise to disk on every call.
- **Hypothesis:** The persistence intent is "remember the user's width across reloads" — that does not require persisting the *intermediate* values during a drag.
- **Suggested next step:** Move the localStorage writes into the `mouseup` handler, or wrap them in a 250 ms trailing-edge debounce. Pairs naturally with F1's "commit on mouseup" change.

### F3 — `DiffViewer` re-commits 300× even though its inputs do not change during resize

- **What:** During phases E and F (resize), the diff data and view mode were unchanged, yet `DiffViewer` commits 300 times.
- **Evidence:** DiffViewer's avg commit (149 µs) is cheap, but its commit count is implausibly high relative to actual prop changes — it's tracking SandboxDetail's re-renders 1:1.
- **Hypothesis:** `SandboxDetail.tsx:859–876` constructs the `DiffViewer` props inline. None of the callbacks (`onDiscardFile`, `onSelectedFileIdsChange`, etc.) are stable across SandboxDetail commits because `SandboxDetail` itself is rebuilt on every parent commit (F1). `DiffViewer` is not wrapped in `memo`, so it re-renders unconditionally.
- **Suggested next step:** Wrap `DiffViewerImpl` (the inner function added in Phase 1 of this audit) with `React.memo`. Then make sure the callback props are stable: hoist `useCallback` for `onDiscardFile` etc. in App.tsx and pass them straight through. The Profiler boundary will reveal whether memoization holds (commit count should drop from 300 to ~10 during a resize).

### F4 — `useSandboxes` polls every 10 s and returns a fresh array reference

- **What:** `lib/hooks.ts:110` sets `refetchInterval: 10000`. Every poll returns a fresh `Sandbox[]` reference even when the underlying list is unchanged. App passes that reference to multiple consumers (`Sidebar`, `ActiveTab`, `useMemo` dependencies in `App.tsx`, plus `mobile`/`desktop` branches).
- **Evidence:** With the audited dataset (9 sandboxes, 12 s window), one poll fired during capture; correlates with one `App` commit at ~10 s into the trace. The cost is currently invisible because the dataset is small, but the same array reference threads through every memo selector below — every poll currently invalidates them.
- **Hypothesis:** TanStack Query gives back a new array on each refetch by design. The downstream code assumes referential equality is *not* a contract, which is true for correctness but silently disables every `useMemo` keyed on `sandboxes`.
- **Suggested next step:** Add a `select` to the `useQuery` that performs structural-equality memoization (e.g. compare `id+updatedAt` per row) and returns the previous reference when nothing changed. Or move the polling source through a `useSyncExternalStore`-style cache that does the dedupe centrally. Validate by capturing during a 30+ s window covering several polls and confirming non-resize-driven App commits are flat.

### F5 — `SandboxItem` is not memoized; list scales linearly with parent commit count

- **What:** `components/sidebar/SandboxItem.tsx:81` exports a plain function component. `ActiveTab.tsx:127–138` and `HistoryTab.tsx:112–125` map over visible sandboxes and render one item per row. With 9+8 rows currently this is invisible; with 100+ rows each App commit triggers 100+ child commits.
- **Evidence:** HistoryTab itself only commits 5 times in the capture, but each commit walks 8 rows. With the resize-driven 369 App commits in F1, a 100-row list would mean ~37 000 row-commits per drag.
- **Hypothesis:** No memoization barrier between the list and its rows. Parent re-renders propagate even when the row's own props (`sandbox`, `selected`, `onSelect`) didn't change.
- **Suggested next step:** Wrap `SandboxItem` in `React.memo`. Pair with stable `onSelect` callbacks (currently `App.tsx` defines them via `useCallback`, good — but verify the `consolidatedMessages: Set<string>` reference passed by `ActiveTab.tsx:134` is stable; today `useBannerData` may rebuild it each render). The reward becomes visible only at scale, but is a prerequisite for F4's polling not to thrash a large list.

### F6 — `useHighlighting` re-runs synchronously on every prop-change pulse, with no language preload guarantee

- **What:** `components/DiffViewer.tsx:195–229`. The hook calls `highlightCode(content, language)` inside a `useEffect` keyed on `[content, filePath]`. The first call pays for shiki's WASM init (~50–150 ms) plus per-language parser load.
- **Evidence:** Not measured in this trace — no archive in the dataset has `viewMode !== 'diff'`, and `ClosedSandboxDetail.tsx:166` never passes `viewMode`, so the hook never fires. Static finding only.
- **Hypothesis:** Two latent issues:
  1. The hook depends on `content` and `filePath` directly. If a parent re-render produces a new `content` string with the same characters (it shouldn't here, but `parseUnifiedDiff` is rebuilt whenever `diff?.unifiedDiff` reference changes — see F4), the hook reruns and pays the highlight cost again.
  2. The `bundledLanguages` array in `lib/highlighter.ts:121–139` eagerly preloads 17 languages into the initial bundle. The `extensionToLanguage` map references many more (e.g. `rust`, `kotlin`, `swift`) that fall through to `loadLanguage` — that path is async and adds first-paint delay when the user opens such a file.
- **Suggested next step:** (a) Memoize highlight results by `(language, hash(content))` outside React state — a simple `Map<string, HighlightedLine[]>` keyed on a content hash, or `useDeferredValue` on `content` so React schedules the work without blocking. (b) Cut `bundledLanguages` to a true "always-loaded" set (`javascript`, `typescript`, `tsx`, `json`, `markdown`, `bash` — 6 entries) and rely on `loadLanguage` for the rest. Measure first-paint impact with a comparison run.

### F7 — `DiffViewer` source/full-file paths render one DOM node per line without virtualization

- **What:** `components/DiffViewer.tsx:299–358` (`FullFileView`, `SourceView`) iterate every line of the file and emit one `<HighlightedCodeLine>` per row. Each row renders a `<div>` with a `<pre>` and inline tokens.
- **Evidence:** Not exercised by the captured dataset (see F6). For a 5 000-line markdown or source file this would emit 5 000+ DOM nodes plus the corresponding token spans.
- **Hypothesis:** The component shape was inherited from the diff-view path (where hunks are typically small) and was extended to source/full-file modes without adding a virtualization barrier.
- **Suggested next step:** Add a windowed renderer (e.g. `@tanstack/react-virtual` — already used elsewhere in Vrooli scenarios per the perf-audit skill's example). Wrap the source/full-file body in a fixed-row-height virtualizer. Diff-mode hunks can stay direct since they're already chunked.

### F8 — `ResizeObserver` reflow re-clamp lives in `SandboxDetail` but depends on `headerHeight`

- **What:** `SandboxDetail.tsx:296–312`. The clamp `useEffect` re-attaches a `ResizeObserver` every time `headerHeight` changes (it's in the deps array).
- **Evidence:** During the drag in phase F, this effect tears down and rebuilds the observer ~150 times in 3 seconds.
- **Hypothesis:** The observer should be installed once on mount; it reads `containerRef.current.clientHeight` lazily, so it doesn't need `headerHeight` in deps. The clamp also calls `setHeaderHeight` from inside the observer callback, which then re-fires the effect — a minor feedback loop.
- **Suggested next step:** Remove `headerHeight` from the deps array and read the latest value via a ref inside the clamp body. Bonus: if F1's "commit on mouseup" pattern ships, the clamp will only need to fire on real container changes, not on every drag pulse.

## Recommendations + outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | F1 — Drive sidebar/header sizing via CSS custom property + RAF, commit only on mouseup | open | Highest leverage. Should drop App commit count from ~370 to ~5 in this scenario. |
| 2 | F2 — Persist localStorage on mouseup, not on every step | open | Bundles naturally with #1. |
| 3 | F3 — Wrap `DiffViewerImpl` in `React.memo`, stabilise its callback props | open | Profiler boundary already in place; verify with comparison run. |
| 4 | F4 — Memoize `useSandboxes` `select`-side to return stable references when row set unchanged | open | Currently invisible at this dataset; load-bearing for any scaling. |
| 5 | F5 — Wrap `SandboxItem` in `React.memo`; verify `consolidatedMessages` set is stable | open | Pairs with #4. |
| 6 | F6 — Cut shiki preloaded language list to ~6; add result-cache for highlighter | open | Run a comparison capture against a sandbox containing files in `source` view to validate. |
| 7 | F7 — Virtualize `FullFileView` / `SourceView` body | open | Out of scope for diff-mode; only relevant when source/full-file modes are exercised. |
| 8 | F8 — Reattach `ResizeObserver` only on mount, not on `headerHeight` change | open | Small win on its own; cleans up after #1 lands. |

## New dependencies

- `@tanstack/react-virtual` (only required by F7 if accepted; the rest of this audit's recommendations need no new packages).

## Re-run checklist

To validate any fix:

```bash
mv /tmp/workspace-sandbox/perf/trace.json /tmp/workspace-sandbox/perf/trace.before.json
mv /tmp/workspace-sandbox/perf/trace.web-vitals.json /tmp/workspace-sandbox/perf/trace.before.web-vitals.json
VROOLI_BUILD_MODE=profile vrooli scenario restart workspace-sandbox
node /tmp/workspace-sandbox/perf/capture.js http://localhost:21239 /tmp/workspace-sandbox/perf/trace.after.json
# then run the Phase 5 / comparison aggregator from scenario-performance-audit SKILL.md
```

Watch `App / update` count and `avg(µs)` — F1 should drop the count by ~70 % at the resize phases. F3's win is visible as `DiffViewer` count tracking only true diff-prop changes (≤10) instead of cascading from SandboxDetail.
