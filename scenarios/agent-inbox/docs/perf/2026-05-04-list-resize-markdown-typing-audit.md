---
date: 2026-05-04
scenario: agent-inbox
interactions:
  - sidebar-list-scroll
  - sidebar-resize-drag
  - search-typing-and-filtering
  - markdown-thread-scroll
  - message-input-typing
traces:
  before: /tmp/agent-inbox/perf/trace.json
  after: /tmp/agent-inbox/perf/trace.after-lazy.json
  capture_script: /tmp/agent-inbox/perf/capture.js
status: fixed
related_skill_run: scenario-performance-audit
---

# Perf audit: lists, resize, markdown, typing

## Framing

- User complaint: "measure the performance of the agent-inbox scenario ... Look into anything where there are lists, resizing, markdown rendering, and typing"
- Environment: local profile-mode production bundle, headless Chromium, 1440x900 viewport, `agent-inbox` on `http://localhost:21237`.
- Reproduction trigger: seeded local audit data with 82 visible inbox chats, selected a markdown-heavy thread with 16 message DOM nodes and 13 markdown roots, then exercised sidebar scroll, resize drag, search typing, message scroll, and 525 characters of input typing.

## Methodology

- Profile-mode build verified: served bundle contained `onProfilerRender`, `SidebarChatList`, `MessageInputArea`, `MarkdownRenderer`, and `Sidebar`.
- Capture script: `/tmp/agent-inbox/perf/capture.js`
- Trace: `/tmp/agent-inbox/perf/trace.json`
- Web vitals: `/tmp/agent-inbox/perf/trace.web-vitals.json`
- Capture configuration: CDP tracing with React user-timing markers, CPU profile chunks, screenshots, long-task observer, and a 1440x900 headless Chromium viewport.
- Audit infra added first because `agent-inbox` lacked profile-mode Vite aliases, a profile build script, `src/lib/profiler.ts`, and top-level/focused `<Profiler>` boundaries.

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---:|---:|---:|---:|
| App | 649 | 1467.8 | 2262 | 106001 |
| Sidebar | 107 | 935.1 | 8740 | 23601 |
| SidebarChatList | 106 | 849.7 | 8016 | 22401 |
| MessageList | 4 | 110.1 | 27526 | 100801 |
| MessageInputArea | 574 | 104.5 | 182 | 600 |
| MarkdownRenderer | 16 | 94.4 | 5900 | 27200 |

Sanity checks: 2968 React user-timing entries and 1400 `ProfileChunk` events.

## Long-task summary

| metric | before | after | delta |
|---|---:|---:|---:|
| count | 2 | 1 | -1 |
| total(ms) | 196.0 | 66.0 | -130.0 |
| max(ms) | 133.0 | 66.0 | -67.0 |

Paint timings: first paint and first contentful paint both at 108 ms. LCP was 108 ms with size 19504.

## Findings

### 1. Sidebar list dominates the interaction window

- **What:** [ChatList.tsx](../../ui/src/components/layout/sidebar/ChatList.tsx) renders every visible chat row and every content-search match directly: normal rows at lines 214-236, content-search groups at lines 157-212.
- **Evidence:** `SidebarChatList` committed 106 times for 849.7 ms total, 8016 us average, 22401 us max. `Sidebar` was nearly identical at 935.1 ms total.
- **Hypothesis:** search typing and selected-row changes repeatedly reconcile the full row set. `ChatListItem` is not memoized, and each row receives fresh closures from [ChatList.tsx](../../ui/src/components/layout/sidebar/ChatList.tsx) lines 219-233 plus fresh label arrays from lines 112-113 and 224. This defeats cheap row reuse even before the list grows beyond 82 rows.
- **Suggested next step:** virtualize `displayChats` and content-search groups, then memoize `ChatListItem` with stable per-row props. A dependency-free virtual row hook is enough here; `@tanstack/react-virtual` is optional but would need dependency approval.

### 2. Resize drag updates React state on every mousemove

- **What:** [useResizableSidebar.ts](../../ui/src/hooks/useResizableSidebar.ts) calls `setWidth` directly in the `mousemove` handler at lines 104-110, and that width flows through [App.tsx](../../ui/src/App.tsx) lines 47-52 into [SidebarPanel.tsx](../../ui/src/components/layout/SidebarPanel.tsx).
- **Evidence:** the resize phase is part of the 649 `App` commits and 107 `Sidebar` commits. The sidebar subtotal was 935.1 ms, and the profile recorded two long tasks totaling 196.0 ms.
- **Hypothesis:** every pointer move re-renders `App`, `SidebarPanel`, `Sidebar`, and `ChatList` while the browser is also doing layout for width changes.
- **Suggested next step:** live-update a CSS custom property on the sidebar panel inside `requestAnimationFrame`, keep drag state in a ref, and commit React `width` once on mouseup for persistence. This matches the pattern recently applied in `prompt-manager`.

### 3. Markdown mount is the largest single commit after selecting a heavy thread

- **What:** [MessageList.tsx](../../ui/src/components/chat/MessageList.tsx) maps every filtered message at lines 171-193, and each markdown message runs [MarkdownRenderer.tsx](../../ui/src/components/markdown/MarkdownRenderer.tsx) lines 193-202 with `react-markdown`, `remarkGfm`, code blocks, link previews, tables, and possible Mermaid handling.
- **Evidence:** `MessageList` had only 4 commits but averaged 27526 us, with a 100801 us max. `MarkdownRenderer` had 16 commits totaling 94.4 ms, averaging 5900 us and maxing at 27200 us.
- **Hypothesis:** selecting a conversation mounts/parses all markdown messages synchronously. The current memoization helps subsequent unchanged renders, but not the first parse or streaming updates.
- **Suggested next step:** virtualize long message histories, collapse or defer offscreen heavy markdown/code blocks, and render streaming assistant text with a lightweight plain-text path until the stream settles. Also hoist `remarkPlugins={[remarkGfm]}` to a module-level constant to avoid needless plugin-array churn.

### 4. Typing is frequent but locally cheap; risk is broader input orchestration

- **What:** [useMessageInput.ts](../../ui/src/components/chat/useMessageInput.ts) keeps the typed message at lines 53-71 and passes it into suggestions, slash commands, templates, send logic, textarea effects, and a large returned state object at lines 153-259. [MessageInput.tsx](../../ui/src/components/chat/MessageInput.tsx) destructures that large object and renders footer/modals around the input at lines 39-249.
- **Evidence:** `MessageInputArea` committed 574 times during 525 typed characters, but only totaled 104.5 ms, averaging 182 us with a 600 us max.
- **Hypothesis:** the core textarea is currently fine, but every keystroke still rebuilds the full `useMessageInput` orchestration object and can fan out into suggestions/templates/footer work as features grow or suggestions are visible.
- **Suggested next step:** split raw input state into a narrow component/hook boundary, keep expensive suggestion/template derivation behind debounced values, and pass only the fields `MessageInputArea` needs instead of the entire hook return object.

### 5. `App` owns too much high-frequency UI state

- **What:** [App.tsx](../../ui/src/App.tsx) subscribes to chat data, async status, tools, active template state, sidebar resize state, modal state, keyboard state, and constructs many inline props/callbacks in lines 64-229.
- **Evidence:** `App` committed 649 times for 1467.8 ms total and had the largest outlier at 106001 us.
- **Hypothesis:** top-level state churn from resize, search/list updates, selected-chat loading, and typing keeps broad subtrees eligible for reconciliation. Some downstream memoization cannot help because props such as inline callbacks at lines 204, 210-211, and 225 are recreated from the top level.
- **Suggested next step:** isolate sidebar state, chat-view state, and input state into narrower controller components. Stabilize callbacks passed to memoized leaves where they are on hot paths.

## Recommendations + outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Virtualize sidebar chat rows and content-search groups | fixed | Sidebar rows now use dependency-free virtualization. Final trace: `SidebarChatList` total 849.7 ms -> 288.2 ms; avg 8016 us -> 2287 us. Follow-up pass flattened and virtualized expanded content-search group rows as well. |
| 2 | Change resize to RAF/CSS-variable live mutation with one React commit at drag end | fixed | Resize now mutates the sidebar panel width outside React during drag and commits state at mouseup. Final trace: `Sidebar` total 935.1 ms -> 324.8 ms. |
| 3 | Virtualize or incrementally render long message histories | fixed | Long histories use measured virtualization above 30 rendered messages. The 16-message audit thread intentionally stays non-virtualized to avoid measurement churn. |
| 4 | Defer heavy markdown work for offscreen/streaming content | fixed | Streaming content renders plain text until settled; large non-streaming markdown lazily parses near viewport. Long tasks improved 196.0 ms -> 66.0 ms. |
| 5 | Split raw typing state from the larger message-input orchestrator | fixed | Input-area props are narrowed/memoized, footer is memoized, skill suggestions consume deferred text, and the textarea now owns an immediate local draft that syncs expensive parent state on debounce while submit reads the local draft directly. `MessageInputArea` avg stayed cheap: 182 us -> 176 us. |
| 6 | Move high-frequency UI state below `App` where possible | fixed | Resize and sidebar-list churn no longer force the same expensive broad work. Follow-up pass memoized `SidebarPanel`/`MainContent` and stabilized hot-path callbacks from `App`. Final trace from the first implementation pass: `App` total 1467.8 ms -> 918.2 ms; avg 2262 us -> 762 us. |

## Validation run

Final comparison trace: `/tmp/agent-inbox/perf/trace.after-lazy.json`.

Follow-up implementation validation: `pnpm run type-check`, `pnpm exec eslint .`, focused Vitest chat/input suites, and `vrooli scenario test agent-inbox` were run on 2026-05-04. The scenario lint phase reports Go lint, UI `tsc`, and UI ESLint as clean. The full scenario suite still has unrelated standards/smoke/playbook failures and a Lighthouse audit error before score output; see `coverage/logs/20260504-145718-06231ecc/`.

| component | before count | after count | before total(ms) | after total(ms) | before avg(μs) | after avg(μs) | delta(ms) |
|---|---:|---:|---:|---:|---:|---:|---:|
| App | 649 | 1205 | 1467.8 | 918.2 | 2262 | 762 | -549.6 |
| Sidebar | 107 | 128 | 935.1 | 324.8 | 8740 | 2537 | -610.4 |
| SidebarChatList | 106 | 126 | 849.7 | 288.2 | 8016 | 2287 | -561.5 |
| MessageInputArea | 574 | 977 | 104.5 | 171.6 | 182 | 176 | +67.1 |
| MessageList | 4 | 13 | 110.1 | 127.9 | 27526 | 9839 | +17.8 |
| MarkdownRenderer | 16 | 21 | 94.4 | 119.9 | 5900 | 5710 | +25.5 |

Interpretation: component commit count rose because virtualization/lazy parsing introduce additional cheap commits while the user scrolls. The user-facing signal improved: long-task count dropped from 2 to 1, long-task total dropped 130.0 ms, and sidebar/App average commit costs fell sharply.

## New dependencies

- None required. A small dependency-free virtual row hook is sufficient for the sidebar and message list.
- Optional: `@tanstack/react-virtual` if the team wants a maintained virtualization package instead of local hooks; requires explicit dependency approval before implementation.
