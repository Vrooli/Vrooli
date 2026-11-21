# Session transcript: typing and diagram flicker

**Scenario:** swarm-manager · **Flow:** `bas/flows/session-transcript-typing.json` · **Date:** 2026-08-10

Reported symptom: a message containing a mermaid diagram flickered between a
loading state and the rendered diagram.

## Root cause

Not mermaid. The whole markdown subtree was being **unmounted and remounted**,
and mermaid re-ran its full layout each time.

`MarkdownRenderer` listed its callers' callbacks in the dependency array for its
`components` map. Every caller passes at least one inline arrow, so the map was
rebuilt on every render — and a rebuilt map means a new `code` *function*, which
React reads as a changed element **type** and answers by discarding the subtree
and mounting a fresh one.

Two things then drove that render on a loop: the session polls every 3s, and
every composer keystroke sets `draft` state on `SessionDetailsPage`.

`ui/src/components/markdown/MarkdownRenderer.tsx:93` (pre-fix dependency array)

## Method

`performance-health audit run swarm-manager --workflow session-transcript-typing`
— production (profile-mode) build, opens a session whose transcript holds a
mermaid diagram, waits for the diagram to settle, then types 67 characters at
40ms intervals. A `MutationObserver` on the **transcript pane** (not the diagram
host — a remount destroys the host, detaching an observer attached to it) counts
skeleton re-appearances and SVG teardowns.

A/B by reverting only the three render-path fixes, everything else held constant.
Two independent captures per arm.

## Results

| Metric | Before | After | Δ |
|---|---|---|---|
| **Skeleton re-appearances** (67 keystrokes) | **210** | **0** | eliminated |
| SVG teardowns | 3 | 0 | eliminated |
| AppShell commits | 168 | 113 | −33% |
| AppShell avg commit | 3.6ms | 2.4ms | −33% |
| Outlet commits | 149 | 93 | −38% |
| Outlet avg commit | 3.4ms | 1.9ms | −44% |
| EventDispatch count | 1732 | 380 | −78% |
| EventDispatch total | 783.1ms | 236.1ms | −70% |
| FunctionCall count | 5132 | 1648 | −68% |
| UpdateLayoutTree total | 285.0ms | 63.5ms | −78% |
| Paint count | 1407 | 465 | −67% |
| Typing overhead above the scripted 2680ms delay | 758ms | 256ms | −66% |

210 skeleton appearances ÷ 67 keystrokes ≈ 3 per keystroke — the observer counts
a remount's added container and its skeleton child, so this is **one full
remount per keystroke**. The diagram never finished rendering while the user
typed, because each keystroke restarted its 120ms debounce.

Reproducibility across the two captures per arm: AppShell commits 168/170 before
vs 110/113 after; Outlet 149/152 vs 91/93. `RunTask` total varied 5273ms vs
9497ms **within** the before arm, so that axis is too noisy to attribute and is
excluded from the table.

## Changes

| File | Change |
|---|---|
| `markdown/MarkdownRenderer.tsx` | Callbacks routed through a ref so `components` memoizes on `[]` — stops the remount. Parsed tree memoized on content. |
| `markdown/useMermaidSvg.ts` | Bounded cross-mount render cache seeded during first render; in-flight dedupe; `initialize()` once. |
| `markdown/MermaidDiagram.tsx` | Skeleton replacing a one-line "Rendering diagram…" that also reserved the wrong height. |
| `chat/ChatMessageBubble.tsx` | `memo()` + `useCallback` on both markdown callbacks. |
| `hooks/useAgentMessageTTS.ts` | Returned a fresh object literal every render, defeating bubble memoization. Memoized; depends on the core's stable functions. |
| `session/SessionConversation.tsx` | Render slots `useCallback`'d. |
| `stores/agent-session-store.ts` | `refreshSession` keeps the existing object when a poll returns unchanged data. |
| `hooks/useAgentSessionEvents.ts` | Depended on the whole session object, so every poll tore down the 2.5s events interval and refetched immediately. |
| `pages/SessionDetailsPage.tsx` + 3 inspector panes | Typing re-rendered the events timeline, artifact list, and details pane. Memoized. |

## Regression guard

The deterministic guard is `ui/src/components/markdown/markdown-stability.test.tsx`
and `ui/src/components/chat/chat-thread-rerender.test.tsx` — verified to fail
when the fixes are reverted (9 mermaid renders per 9 parent renders; 6 message
re-renders per no-op poll).

No `performance-health budget` was set. Budgets are fed by the continuous
capture sweep, and the sweep cannot currently run for this scenario — see below.

## Tooling defects found

1. **The `--workflow` capture path rejects the repo's own flow files.**
   `performance-health audit run` sends the whole flow file as
   `interaction_flow_json` and BAS validates it as `WorkflowDefinitionV2`
   (`{metadata, settings, nodes, edges}`), but the checked-in flows are the
   saved-workflow envelope (`id`, `project_id`, `name`, `description`, `tags`,
   `created_at`, `flow_definition`, …). Every existing flow fails:
   `unknown field "createdAt"` / `unknown field "description"`. Either the
   caller should unwrap `flow_definition`, or the files should be bare
   definitions. This flow is stored bare so it captures; BAS rewrites it back to
   the envelope after each successful run, which breaks the next run.
2. **`ACTION_TYPE_INPUT` fills rather than types.** With `delay_ms: 45` the
   trace recorded a single `input` event, so no per-keystroke render pressure.
   This flow drives typing from an `ACTION_TYPE_EVALUATE` loop instead.
3. **The analyzer reports `LCP=0ms FCP=0ms`** while the sibling
   `performance.web-vitals.json` holds `lcp: 216`, `fcp: 128`.
4. **`Frame health duration` is nonsense** — `552581150.0ms` (~6.4 days).
5. **`vite.config.ts:27` points at `scratch/perf-spike/README.md`**, which does
   not exist anywhere in the repo.
   *Resolved 2026-08-30: the README was written on 08-21, then perf-spike was
   retired entirely — BAS's `playwright-driver/src/tracing/performance-tracer.ts`
   supersedes it with the same CDP capture, streamed and non-throwing. The
   comment now points at `scenarios/test-genie/docs/phases/performance/README.md`.*
