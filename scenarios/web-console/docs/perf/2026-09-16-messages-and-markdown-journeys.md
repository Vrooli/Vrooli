---
date: 2026-09-16
scenario: web-console
interactions:
  - perf-view-switch
  - perf-messages-scroll
  - perf-markdown-preview
flows:
  - scenarios/web-console/bas/flows/perf-view-switch.json
  - scenarios/web-console/bas/flows/perf-messages-scroll.json
  - scenarios/web-console/bas/flows/perf-markdown-preview.json
status: fixes-landed-measurement-inconclusive
related_skill_run: performance
---

# Perf audit round 2: view switch, messages scroll, large markdown

Follows `2026-09-15-workspace-steady-state-audit.md`. Operator report: "switching
from the terminal view to the Messages view, scrolling the messages view, and
rendering/scrolling a large markdown file in the file viewer are all slow and
laggy."

## Headline

The causes were found in source and fixed. **The measurement did not
succeed**: every A/B is confounded because the capture drives the operator's
LIVE workspace, and run-to-run background traffic swamps the effect size. Do
not read the deltas below as evidence for or against the fixes.

## What the journeys actually cost (absolute, fixed tree)

| Journey | Workspace commits | MessagesPane | Long tasks | Notes |
|---|---:|---:|---:|---|
| View switch + 6s hold | 107–191 | 20–91 | 263–304 ms | first rows paint 182–187 ms after the click |
| Scroll 90 up + 90 down | 483–487 | 382–393 | 281–660 ms | ~2 MessagesPane commits per scroll event |
| Open + scroll 156-block .md | 146 | 71 | 504 ms | open 912 ms; 19 fences; 18,219 px tall |

The switch itself is **not** slow (first rows in ~185 ms). The lag is the
sustained re-render storm while Messages is open, and the per-scroll-event
measurement loop.

## Causes found (file:line) and what was done

1. **Row ref was an inline arrow** — `MessagesPane.tsx:918` (old numbering).
   React re-attached every row ref on every commit, so `useVirtualList`
   (`hooks/useVirtualList.ts:224-268`) disconnected and rebuilt that row's
   ResizeObserver and forced two `getBoundingClientRect` reads **per row, per
   scroll event**. Fixed: `registerItem` no longer changes identity (volatile
   deps moved to refs) and the hook exposes cached per-index `itemRef(index)`.
2. **`MessageRow`'s memo could never return true** — `MessagesPane.tsx:780-819`
   rebuilt `onRewind` and `onSaveAsSnippet` closures per row per render, and
   the comparator (`components/messages/MessageRow.tsx:310-329`) compares the
   action context entry by entry. Fixed: `onRewind` hoisted to a stable
   callback; `onSaveAsSnippet` cached per event id.
3. **Layout read in a render body** — `components/tts/PlaybackModeControl.tsx:85`
   called `getBoundingClientRect()` unconditionally, once per assistant row.
   Fixed: gated on `open`.
4. **Scroll state updated per event** — `useVirtualList.ts:135`. Fixed:
   coalesced to one `setScrollTop` per frame (with cancel on cleanup).
5. **Navigation builders scanned every event of every pane on every render** —
   `Workspace.tsx` navigation/sidebar/unread builders. Fixed: memoized behind an
   O(panes) signature built from the store's maintained `maxSequence` and
   `unreadCount`, plus a one-per-minute tick so relative labels still update.
6. **Two shiki engines** — `markdown/components/CodeBlock.tsx:29` built its own
   highlighter with a different language set from `lib/codeHighlighter.ts`.
   Fixed: one singleton, languages merged.
7. **shiki in the always-loaded chunk** — `lib/codeHighlighter.ts:1` imported it
   statically, defeating `CodeBlock`'s dynamic import. Fixed: type-only import +
   dynamic `import("shiki")`. Workspace chunk 2,119,290 → 1,983,487 B; engine
   now a separate 190,887 B chunk (verified: zero `oniguruma`/`createOnigScanner`
   markers remain in the Workspace chunk).
8. **Every fence highlighted in one tick on open** — Fixed: `CodeBlock` gates
   highlighting on an IntersectionObserver (800 px margin); falls open where
   IntersectionObserver is unavailable.
9. **`detectLanguage` (~40 regexes over the block body) ran per render** —
   Fixed: memoized.
10. **Whole markdown document laid out and painted** — Fixed (partially): a
    scoped `content-visibility: auto` rule in `styles.css` for
    `[data-testid="file-preview-markdown"] .markdown-content > *`.

## Why the measurement failed

- `perf-messages-scroll`: final scrollTop 44,203 vs 6,351 between runs, RunTask
  count 68,988 → 100,438. Different journeys; totals not comparable.
- `perf-view-switch`: flow shape matched (msToFirstRows 187 vs 182), but
  **TerminalPane commits fell 105 → 23**. Nothing in this round can affect
  TerminalPane; its commits track terminal output. The second capture simply had
  far less live traffic, which deflates every component.
- Conclusion: a valid A/B needs a quiescent target (an isolated instance with no
  live sessions) or many repetitions with median comparison. Against the
  operator's working console, variance >> effect.

## Still open

1. **`Workspace` subscribes to the whole conversation `sessions` map**
   (`Workspace.tsx:287`). This is the render-COUNT driver and it is unfixed.
   `useTtsPlaybackController` runs inside `Workspace` and memoizes
   `derivedState`/`pillContext` on the map's identity, indexing it by arbitrary
   session id for queue building (`useTtsPlaybackController.ts:388,657,675,679`).
   Dropping the map staless playback state; a hook that subscribes internally
   still re-renders `Workspace`. The real fix is moving the playback controller
   and its consumers below the root — a restructuring, not a local edit.
2. **~2 MessagesPane commits per scroll event** remain, from the
   measure → `sizeVersion` → geometry-rebuild → re-render loop
   (`useVirtualList.ts:198-222`, rebuild is O(count) from the dirty index).
3. **CLS 2.4–3.1 with ~90 shift entries while scrolling the message list**,
   present before and after this round. The list visibly shifts as rows measure.
   Strong candidate for the felt lag; unaddressed.
4. **CLS 1.180 in the markdown preview.** `contain-intrinsic-size: auto 120px`
   guesses a height for every off-screen block; wrong guesses become shifts. No
   pre-change capture of this journey exists, so my own change cannot be
   exonerated here.

## Incidents worth remembering

- I introduced a **hooks-after-early-return** bug: `Workspace` returns early at
  `!isHydrated` and at zero panes, and the memo block was inserted below those
  returns. The full test suite did **not** catch it (identical 39 failures
  before and after the fix); `eslint react-hooks/rules-of-hooks` did.
- The UI suite is **not green at baseline**: 39 failed / 2,619 passed / 269
  files, unchanged by this round's work (zero new, zero resolved).
- `messagesFileViewer.linePrefix` exists in all locale catalogs and the
  generated key constants but is rendered by nothing — filed as
  `knw-1789519374020043683`.
