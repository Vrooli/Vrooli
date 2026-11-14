---
date: 2026-09-15
scenario: web-console
interactions:
  - workspace-steady-state-hold
traces:
  before: /home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/captures/9687ee12-f2a4-4635-9d38-f8045cf008b8/19e5b001-84b7-4f13-920e-f96e39c8d1f2/performance/performance.json
  after: /home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/captures/3833bfd5-1702-435b-a74a-c2b64e98f2e6/46c76bce-c073-4d52-acc7-7a548572e46e/performance/performance.json
  flow: scenarios/web-console/bas/flows/perf-terminal-stream.json
status: fixed
related_skill_run: performance
---

# Perf audit: workspace steady state

## Framing

- Request: identify and address UI performance issues in the web console.
- Environment: profile-mode production build, BAS headless Chromium, 1440x900, `web-console` on `http://localhost:21233`.
- Interaction traced: the workspace sitting in **steady state** — mounted, with live sessions attached, no user input for 12 s.

## Two measurement traps hit first

1. **The default capture measures the wrong screen.** `performance-health audit run` with no `--workflow` stops at navigate (396 ms) and settles. `App` lazy-loads `Workspace` behind `<Suspense>`, so the trace captured only the "Loading..." fallback: LCP 124 ms, 2 `App` commits, no network activity, no console output. Any perf flow for this scenario must wait for a mounted `[data-testid="terminal-pane"]`, not just the shell.
2. **Only `App` had a `<Profiler>` boundary**, so every commit below it was unattributable. Boundaries were added for `Workspace`, `TabBar`, `SessionSidebar`, `TerminalPane` and `MessagesPane`. They are inert outside the perf-build channel.

A third trap cost a capture cycle: the first perf flow tried to launch a terminal via `new-terminal-button`/`toolbar-new`. Both test ids exist in the DOM in layouts where the control is **hidden** (`FloatingToolbar` is `hidden` in mobile tab-like mode; `new-terminal-button` belongs to the empty-workspace placeholder), so the wait succeeded and the click silently did nothing. The flow is now read-only.

## Baseline (12 s, idle, no interaction)

| Surface | commits | avg | max |
|---|---:|---:|---:|
| App | 98 | 5.9 ms | 65.3 ms |
| Workspace | 94 | 5.9 ms | 65.3 ms |
| SessionSidebar | 77 | 1.3 ms | 16.4 ms |
| TerminalPane | 12 | 0.9 ms | 4.2 ms |

Frames: 593 drawn, 71 dropped (10%). Long tasks 200 ms. RunTask total 9481.6 ms, max 210.8 ms.

**~8 re-renders per second of the 2615-line workspace root while nobody touches the app.**

## Causes found (file:line, all confirmed in source)

1. **`setViewerCount` on every output frame.** `hooks/terminal/useTerminalSession.ts:399` calls it for *every* WebSocket message, including `stdout`. `stores/useWorkspaceStore.ts:642` built a new `viewerCounts` object unconditionally. The store is wrapped in `persist`, which re-serializes the whole partialized state to `localStorage` on **every** `set` — including one that returns state unchanged. So each output chunk cost a store write, a JSON serialization, a synchronous `localStorage.setItem`, and a re-render of every `useWorkspaceStore` subscriber (`SessionSidebar` selects `viewerCounts` at `components/SessionSidebar.tsx:126`).
2. **Per-pane unread selector scans all events.** `components/WorkspacePaneShell.tsx:118` filtered the entire `session.events` array inside a zustand selector, re-run on every conversation-store write for every mounted pane, ignoring the `unreadCount` the store already maintains.
3. **Navigation builders scan all events per pane.** `lib/workspaceNavigation.ts:206` `countUnreadMessages` filtered every event; it is called by `countWorkspaceUnreadMessages`, `buildWorkspaceNavigationItems` and `buildOriginBucketedNavigation` (lines 215, 280, 322, 540) — once per pane, on every render of the workspace root.

## Fixes

- `stores/useWorkspaceStore.ts` — `setViewerCount` returns early when the count is unchanged, so no `set`, no persist write, no subscriber notification. Returning the old state from inside `set` would **not** have been enough: persist writes regardless.
- `components/WorkspacePaneShell.tsx` — the unread selector calls `getSessionUnreadCount`, which reads the maintained count (O(1)) and falls back to the filter.
- `lib/workspaceNavigation.ts` — `countUnreadMessages` reads `session.unreadCount ?? <filter>`; the snapshot type gained the optional field.

Regression test: `stores/useWorkspaceStore-actions.test.ts` — "ignores a viewer count that has not changed" asserts zero subscriber notifications **and** zero `Storage.setItem` calls for a repeated count. It fails before the fix (2 notifications) and passes after.

## Result (same flow, same viewport)

| Surface | commits | avg | max |
|---|---|---|---|
| App | 98 → 66 (−33%) | 5.9 → 6.3 ms | 65.3 → 53.9 ms |
| Workspace | 94 → 60 (−36%) | 5.9 → 6.7 ms | 65.3 → 53.9 ms |
| SessionSidebar | 77 → 43 (−44%) | 1.3 → 1.2 ms | 16.4 → 10.8 ms |
| TerminalPane | 12 → 8 | 0.9 → 1.3 ms | 4.2 → 4.2 ms |

- Dropped frames 71 → 43. Long tasks Δ−29 ms. RunTask total 9481.6 → 8193.0 ms (Δ−1288.6 ms), max 210.8 → 146.3 ms. RasterTask total Δ−93.6 ms.
- **Avg per commit rose slightly** (Workspace 5.9 → 6.7 ms). Expected: the removed commits were the cheap redundant ones, so the remaining population is more substantive. The honest headline is commit *count* and long-task/RunTask totals.
- **LCP 2720 → 3356 ms and CLS 0.008 → 0.208 are NOT comparable between these two runs.** The capture drives the operator's live workspace, whose sessions, message volume and archive differ run to run. Neither figure is claimed as a win or a regression.

## Known-real, not addressed in this pass

1. **`Workspace` subscribes to the whole conversation `sessions` map** (`components/Workspace.tsx:286`). Any `appendEvent`/`mergeEvents`/`updateEvent`/`updateCursor` for *any* session re-renders the root, which then rebuilds navigation items, origin buckets, the unread total and the TTS controller's derived state, and re-runs the Media Session effect. This is the largest remaining driver; narrowing it means moving those derivations into children, which is a real refactor rather than a safe local edit.
2. **shiki ships in the always-loaded Workspace chunk.** `components/markdown/components/CodeBlock.tsx:28` lazy-imports shiki, but `lib/codeHighlighter.ts:1` imports `createHighlighter` statically, on the always-reachable chain `MessagesPane.tsx:26` → `MessagesFileViewer.tsx:10` → `file-preview/renderers/index.ts:2` → `TextRenderers.tsx:13`. The dynamic import is therefore defeated. `Workspace-*.js` is 2.1 MB and the entry 944 KB, of 16 MB `dist`.
3. **Voice recording re-renders the root ~15x/s** while the mic is on (`audio-capture-browser` `useVoiceCore.ts:729` sets `audioLevel` every 66 ms; consumed in `Workspace.tsx:1160`). `VoiceMicButton` already draws its waveform imperatively via rAF; the meter could use the same path.
4. **Scroll/drag handlers set state per event** without rAF batching: `hooks/useVirtualList.ts:146` (messages scroll), `Workspace.tsx:1573-1591` (splitter drag, persisted on every pointermove), drag-reorder in `TabBar.tsx`/`SessionSidebar.tsx`.

## Tooling defects observed

- `performance-health audit run` reported `AUDIT_OUTCOME_UNAVAILABLE` with "browser-automation-studio unreachable" when BAS was healthy and the flow had actually failed at a step (`launched pane did not appear`). A flow-step failure must not be reported as capture-mechanism unavailability.
- The perf capture runs against the operator's **live** workspace despite `routed_isolation: "true"` on the flow. A mutating perf flow can therefore create and delete real sessions, and capture artifacts contain live terminal content.
- `drawn-fps` remains broken (0.0 with a ~1.1e9 ms frame duration); no budget may be set from it.
