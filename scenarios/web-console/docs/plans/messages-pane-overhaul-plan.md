# Messages Pane Overhaul — Implementation Plan

## 1. Purpose

Transform the web-console's `MessagesPane` from a plain-text card-based viewer into a full-featured, markdown-rendered, auto-scrolling conversation interface with compact layout, mermaid diagrams, and random-access message navigation.

## 2. Required Reading

```bash
# Skills
prompt-manager skill read implementation-plan-authoring react-coherence seam-discovery-and-enforcement test utils-unification

# Reference implementations (agent-inbox markdown rendering)
cat scenarios/agent-inbox/ui/src/components/markdown/MarkdownRenderer.tsx
cat scenarios/agent-inbox/ui/src/components/markdown/components/CodeBlock.tsx
cat scenarios/agent-inbox/ui/src/components/markdown/components/MermaidDiagram.tsx
cat scenarios/agent-inbox/ui/src/components/markdown/components/InlineCode.tsx
cat scenarios/agent-inbox/ui/src/components/markdown/hooks/useCodeCopy.ts
cat scenarios/agent-inbox/ui/src/components/markdown/utils/languageDetection.ts

# Current implementation
cat scenarios/web-console/ui/src/components/MessagesPane.tsx
cat scenarios/web-console/ui/src/components/MessagesSearchDrawer.tsx
cat scenarios/web-console/ui/src/__tests__/MessagesPane.test.tsx
cat scenarios/web-console/ui/package.json
```

## 3. Problem Statement

The current `MessagesPane` has five issues:

1. **Plain text rendering** — Agent messages are displayed as raw text with `whitespace-pre-wrap`. Code blocks, headings, tables, lists, mermaid diagrams, and other markdown elements are not rendered. This is the primary content format for coding agents.

2. **Wasted horizontal space** — Card-based bubble layout with `rounded-2xl`, `max-w-[85%]`, and `ml-auto` wastes significant screen width, especially on mobile. For a code-agent conversation viewer, this is the wrong tradeoff.

3. **No auto-scroll** — The scroll container renders top-down and lands at the top of the conversation. Users must manually scroll to the bottom every time. No "new messages" indicator when scrolled up.

4. **No random-access navigation** — The chevron up/down buttons only step through messages sequentially. For long conversations (50+ messages), there's no way to jump to a specific message.

5. **No message collapsing** — Long agent messages (common with coding agents) create walls of text that make it hard to scan the conversation for a specific exchange.

## 4. Scope

### In scope
- Markdown rendering with syntax highlighting, mermaid diagrams, GFM tables, task lists
- Full-width, accent-bar message layout replacing card bubbles
- Auto-scroll to newest message + "new messages" pill when scrolled up
- Message jump dropdown for random-access navigation
- Collapsible long messages with expand/collapse toggle
- Search highlighting within rendered markdown
- All associated tests
- SEAMS.md updates

### Out of scope
- Changes to the conversation store data model
- Changes to the TTS/audio system (existing props preserved)
- Changes to the Workspace integration layer
- Changes to the backend API
- Changes to MobileToolbar beyond what's needed for the new layout
- Link preview (agent-inbox has `LinkWithPreview` but it requires a metadata API endpoint — skip for now)

## 5. Current Technical Context

### Key Files

| File | Role |
|------|------|
| `scenarios/web-console/ui/src/components/MessagesPane.tsx` (547 lines) | Main component — renders messages, search, nav, audio popovers |
| `scenarios/web-console/ui/src/components/MessagesSearchDrawer.tsx` (128 lines) | Bottom-sheet search UI |
| `scenarios/web-console/ui/src/__tests__/MessagesPane.test.tsx` | Existing tests |
| `scenarios/web-console/ui/src/stores/useConversationStore.ts` | Session/event state |
| `scenarios/web-console/ui/src/stores/useWorkspaceStore.ts` | Pane layout/settings |
| `scenarios/web-console/ui/package.json` | Dependencies |

### Reference Implementation

The **agent-inbox** scenario has the most complete markdown rendering stack:
- `react-markdown` + `remark-gfm` for parsing
- `shiki` for syntax highlighting (lazy-loaded singleton)
- `mermaid` for diagram rendering (lazy-loaded singleton, 100ms debounce)
- Custom components: `CodeBlock`, `InlineCode`, `MermaidDiagram`, `LinkWithPreview`
- Error boundary for graceful fallback
- Memoized component map to prevent re-parses

This implementation is well-tested and proven. We will adapt it for web-console, using the same library stack and patterns but with web-console's `wc-*` design token classes instead of agent-inbox's hardcoded slate classes.

### Styling Convention

Web-console uses semantic CSS custom properties prefixed `wc-`:
- `text-wc-text-primary`, `text-wc-text-muted`, `text-wc-text-faint`
- `bg-wc-surface`, `bg-wc-surface-base`, `bg-wc-surface-raised`
- `border-wc-default`, `border-wc-accent`
- `bg-wc-accent/10`, `bg-wc-accent/20`, etc.

All new components must use these tokens, not hardcoded slate/indigo values.

## 6. Target End State

### Layout

Messages are displayed as full-width items separated by subtle horizontal dividers. Each message has:
- A thin left accent bar (3px) — one color for user messages, another for assistant
- A compact header row: role label, source, sequence number, action buttons (copy, play, audio)
- Rendered markdown content (or plain text fallback on error)
- A collapse/expand toggle for messages exceeding ~15 lines of rendered content

### Markdown Rendering

All markdown content is rendered with:
- GFM support (tables, task lists, strikethrough, autolinks)
- Shiki syntax highlighting for code blocks (21 languages, `github-dark` theme)
- Mermaid diagram rendering with source/diagram toggle
- Inline code with monospace styling
- Headings, lists, blockquotes, horizontal rules
- Error boundary with plain-text fallback

### Auto-Scroll

- On initial load: scroll to the bottom (most recent message)
- On new message: if user is near bottom (within 200px), auto-scroll to new message
- If user has scrolled up: show a floating "New messages" pill at the bottom that scrolls down on click
- The pill shows a count of unread messages below the fold

### Message Jump

The control strip includes a clickable "Message N of M" indicator that opens a dropdown/bottom sheet listing all messages. Each entry shows: sequence number, role, truncated text (~50 chars). Clicking an entry scrolls to that message.

### Search

Search highlighting works within rendered markdown by searching the raw text content (not rendered HTML). When search is active, messages that don't match are visually dimmed rather than hidden, preserving conversation context.

## 7. Implementation Strategy

### Phase 1: Markdown Rendering Stack (no layout changes yet)

**Goal:** Replace plain text rendering with markdown, keeping the existing card layout temporarily.

1. **Add dependencies** to `package.json`:
   - `react-markdown` ^9.0.1
   - `remark-gfm` ^4.0.0
   - `shiki` ^1.24.0
   - `mermaid` ^11.4.1

2. **Create markdown component directory** at `src/components/markdown/`:
   ```
   markdown/
   ├── index.ts                    # Barrel exports
   ├── MarkdownRenderer.tsx        # Main renderer (adapted from agent-inbox)
   ├── components/
   │   ├── CodeBlock.tsx           # Syntax highlighting with shiki
   │   ├── InlineCode.tsx          # Inline code styling
   │   └── MermaidDiagram.tsx      # Mermaid diagram renderer
   ├── hooks/
   │   └── useCodeCopy.ts          # Copy-to-clipboard hook
   └── utils/
       └── languageDetection.ts    # Pattern-based language detection
   ```

3. **Adapt agent-inbox components** for web-console:
   - Replace hardcoded `slate-*` / `indigo-*` classes with `wc-*` semantic tokens
   - Remove `LinkWithPreview` (no metadata API)
   - Remove dependency on agent-inbox's `useToast` — use a simple local `copied` state (web-console already does this in MessagesPane)
   - Keep: lazy singleton pattern for shiki/mermaid, error boundary, memoized components, language detection

4. **Integrate into MessagesPane**: Replace the `whitespace-pre-wrap` text div (line 534) with `<MarkdownRenderer content={event.text} />`.

5. **Update search highlighting**: The `highlightMatches` function currently operates on raw text. With markdown rendering, search must operate on the raw `event.text` (not rendered output). Add a `searchQuery` prop to `MarkdownRenderer` so it can apply `<mark>` highlights during rendering via a custom remark plugin or post-processing.

**Seams:**
- `MarkdownRenderer` is a pure presentation seam — it takes `content: string` and renders React nodes. No side effects, no store access.
- `CodeBlock` and `MermaidDiagram` have async side effects (loading shiki/mermaid) isolated behind lazy singletons.
- `useCodeCopy` is a thin clipboard seam.

### Phase 2: Layout Overhaul (full-width accent bars)

**Goal:** Replace card bubbles with compact, full-width messages.

1. **Remove card styling** from `<article>`:
   - Remove: `rounded-2xl`, `shadow-sm`, `px-4 py-3`, `ml-auto max-w-[85%]`, `bg-wc-accent/10`, `bg-wc-surface`
   - Add: `border-l-[3px]` with role-based color, `py-3`, horizontal `border-b border-wc-default` as divider

2. **Simplify header row**:
   - Keep: Copy, Play, Audio buttons, role label, sequence number, summarized badge
   - Remove: `cursor-pointer` on the article (clicking to focus is replaced by message jump)
   - Compact the header to a single tight row

3. **Remove `max-w-3xl`** from the inner container — messages use full available width.

4. **Update control strip**: Remove `pr-12` reservation now that the floating toggle button layout may change.

5. **Adjust TTS active indicator**: Use the left accent bar color change instead of a separate `border-l-[3px] border-l-wc-accent` override.

### Phase 3: Auto-Scroll

**Goal:** Auto-scroll to newest messages with "new messages" pill.

1. **Add scroll container ref** to the outer `div` of MessagesPane.

2. **Add `useEffect` for initial scroll**: After events hydrate, scroll to bottom.

3. **Track "near bottom" state**: Use a scroll event listener (debounced) to track whether user is within 200px of the bottom. Store in a `useRef<boolean>`.

4. **Auto-scroll on new event**: When `events.length` changes and user is near bottom, scroll to bottom smoothly.

5. **"New messages" pill**: When user is NOT near bottom and new events arrive:
   - Show a floating pill at the bottom of the scroll container: "N new messages ↓"
   - Clicking it scrolls to bottom and dismisses the pill
   - The pill auto-dismisses when user manually scrolls to bottom

6. **Implementation detail**: Use `IntersectionObserver` on a sentinel `<div>` at the bottom of the message list to detect "near bottom" state — more performant than scroll event listeners.

### Phase 4: Message Jump Dropdown

**Goal:** Add random-access message navigation.

1. **Add "Message N of M" display** to the control strip, between the search button and chevron buttons.

2. **On click**, open a dropdown (desktop) or bottom sheet (mobile) listing all messages:
   - Each entry: `#N · Role · "truncated text..."` (50 chars max)
   - Current/focused message is highlighted
   - Scrollable list with the current message scrolled into view
   - Clicking an entry calls `focusAndScroll(eventId)` and closes the dropdown

3. **Create `MessageJumpList` component** as a new file. Use portal rendering (same pattern as AudioPopoverContent).

4. **Keyboard support**: When jump list is open, arrow keys navigate entries, Enter selects, Escape closes.

### Phase 5: Collapsible Long Messages

**Goal:** Allow collapsing messages that exceed a height threshold.

1. **Measure rendered content height** using a `ResizeObserver` on each message's content div.

2. **Collapse threshold**: If rendered height exceeds 400px (~15 lines of text), collapse to 400px with a gradient fade-out and "Show more" button.

3. **State**: `collapsedIds: Set<string>` — messages start collapsed by default if they exceed the threshold. User can toggle.

4. **When collapsed**: Apply `max-h-[400px] overflow-hidden` with a gradient overlay (`bg-gradient-to-b from-transparent to-wc-surface-base`) and a "Show more (N lines)" button below.

5. **When expanded**: Show full content with a "Show less" button at the bottom.

6. **Search interaction**: When a search match is found inside a collapsed message, auto-expand it.

### Phase 6: Tests

**Goal:** Comprehensive test coverage for all new functionality.

**Unit tests** (new files in `src/__tests__/`):

1. **`MarkdownRenderer.test.tsx`**:
   - Renders basic markdown (headings, lists, bold, italic)
   - Renders GFM tables
   - Renders code blocks with language class
   - Detects mermaid language and renders MermaidDiagram
   - Falls back to plain text on error (error boundary)
   - Handles empty/null content
   - Handles non-string content

2. **`CodeBlock.test.tsx`**:
   - Renders code with language label
   - Shows copy button, copies to clipboard
   - Falls back to plain text when shiki fails
   - Normalizes language aliases

3. **`MermaidDiagram.test.tsx`**:
   - Renders diagram SVG
   - Toggles between source and diagram view
   - Shows error with source fallback
   - Handles empty code

4. **`languageDetection.test.ts`**:
   - Detects TypeScript, Python, Go, SQL, bash, JSON, etc.
   - Returns "text" for ambiguous input
   - normalizeLanguage maps aliases correctly

5. **Updated `MessagesPane.test.tsx`**:
   - Existing tests updated for new layout (no more card-specific selectors)
   - New: auto-scroll to bottom on mount
   - New: "new messages" pill appears when scrolled up
   - New: message jump dropdown opens and navigates
   - New: long messages are collapsed by default
   - New: collapsed messages expand on search match
   - New: markdown content is rendered (not plain text)

6. **`MessageJumpList.test.tsx`**:
   - Renders list of messages with role and truncated text
   - Clicking entry calls onSelect
   - Keyboard navigation (arrows, Enter, Escape)

**Testing approach:**
- Mock `shiki` and `mermaid` imports (they require async loading) — use `vi.mock()` to return stub highlighter/renderer
- Use `@testing-library/react` for DOM assertions
- Use `renderWithProviders` from existing test-utils
- Test behavior at seam boundaries, not internal implementation

## 8. Contract Decisions

### Props (unchanged)
`MessagesPane` props remain identical — no breaking changes to the Workspace integration:
```typescript
interface MessagesPaneProps {
  sessionId: string;
  onSpeakFromHere: (eventId: string) => void;
  onSpeakOne: (eventId: string, text: string, paragraphs?: string[], opts?: { version?: "active" | "original" }) => void;
  activeSpeakingEventId: string | null;
  isTtsSpeaking: boolean;
}
```

### MarkdownRenderer API
```typescript
interface MarkdownRendererProps {
  content: string;
  className?: string;
  searchQuery?: string;           // Highlights matching text within rendered content
  isSearchFocused?: boolean;      // Whether this message contains the focused search match
}
```

### New data-testid selectors
| Selector | Element |
|----------|---------|
| `msg-jump-trigger` | "Message N of M" button in control strip |
| `msg-jump-list` | Jump list dropdown/sheet container |
| `msg-jump-item-{eventId}` | Individual jump list entries |
| `msg-new-pill` | "New messages" floating pill |
| `msg-collapse-{eventId}` | Collapse/expand toggle button |
| `msg-markdown-{eventId}` | Markdown content wrapper per message |

### CSS Token Usage
All new components use `wc-*` tokens. Mapping from agent-inbox's hardcoded values:

| Agent-inbox | Web-console |
|-------------|-------------|
| `bg-slate-900` | `bg-wc-surface-base` |
| `bg-slate-800` | `bg-wc-surface` |
| `bg-slate-700` | `bg-wc-surface-raised` |
| `border-slate-700` | `border-wc-default` |
| `text-slate-200` | `text-wc-text-primary` |
| `text-slate-400` | `text-wc-text-muted` |
| `text-indigo-500` | `text-wc-accent` |
| `border-indigo-500` | `border-wc-accent` |

## 9. Testing Plan

See Phase 6 above for the full test breakdown. Summary:

| Test File | Coverage Target |
|-----------|----------------|
| `MarkdownRenderer.test.tsx` | Rendering all element types, error boundary, edge cases |
| `CodeBlock.test.tsx` | Highlighting, copy, fallback, language normalization |
| `MermaidDiagram.test.tsx` | Render, toggle, error, empty |
| `languageDetection.test.ts` | Detection accuracy, alias normalization |
| `MessagesPane.test.tsx` | Layout, auto-scroll, pill, jump, collapse, markdown integration |
| `MessageJumpList.test.tsx` | List rendering, selection, keyboard nav |

**Validation commands:**
```bash
cd scenarios/web-console/ui && npx vitest run --reporter verbose
cd scenarios/web-console/ui && npx vitest run --coverage
```

## 10. Rollout / Validation Checklist

- [ ] `npm install` succeeds with new dependencies
- [ ] `npx vitest run` — all tests pass
- [ ] Scenario starts: `vrooli scenario restart web-console`
- [ ] Manual verification: open messages view, send a message with markdown (code block, heading, list)
- [ ] Manual verification: mermaid diagram renders from ` ```mermaid` fence
- [ ] Manual verification: auto-scroll to bottom on new message
- [ ] Manual verification: scroll up, new message arrives, "new messages" pill appears
- [ ] Manual verification: message jump dropdown lists all messages, clicking navigates
- [ ] Manual verification: long messages are collapsed, "Show more" expands
- [ ] Manual verification: search highlights text within markdown
- [ ] No console errors or warnings

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Bundle size increase from shiki + mermaid | Larger initial load | Both are lazy-loaded singletons — zero cost until first code block / mermaid diagram |
| Shiki highlighting is async — flash of unstyled code | Visual flicker | Show plain `<pre>` immediately, swap to highlighted version when ready (current agent-inbox pattern) |
| Mermaid can crash on malformed input | Blank diagram area | Error boundary shows source code + error message (proven in agent-inbox) |
| Search highlighting in markdown is harder than in plain text | Broken highlights | Search operates on raw `event.text`, not rendered DOM. Highlight injection uses a custom text-splitting approach per text node. |
| `ResizeObserver` for collapse may cause layout thrash | Performance | Debounce observer, only measure once on mount + when content changes |
| Scroll auto-detection edge cases | Missed auto-scrolls or unwanted scrolls | Use `IntersectionObserver` on sentinel div — binary signal, no scroll math needed |

## 12. Non-Goals / Prohibited Patterns

- **No compatibility shims** — This is greenfield. No `legacyLayout` flags, no `usePlainText` fallbacks for old code paths, no feature toggles.
- **No dead code** — Remove all replaced code (card bubble styles, plain text rendering, old `highlightMatches` function). Don't comment it out or leave it behind.
- **No `lib/` folders in scenarios** — Use v2.0 `service.json` lifecycle. (This plan doesn't create any.)
- **No hardcoded color values** — All new styling uses `wc-*` semantic tokens.
- **No link preview** — `LinkWithPreview` requires a metadata API. Links render as styled `<a>` tags.
- **No changes to store schema** — All new state is component-local.
- **No changes to `MessagesPaneProps`** — The Workspace integration contract is stable.

## 13. Definition of Done

- [ ] All markdown elements render correctly: headings (h1-h4), paragraphs, bold/italic/strikethrough, ordered/unordered lists, task lists, blockquotes, horizontal rules, tables, code blocks (with syntax highlighting), inline code, mermaid diagrams
- [ ] Code blocks show language label, copy button, and shiki-highlighted code
- [ ] Mermaid diagrams render SVG with source/diagram toggle and error fallback
- [ ] Messages use full-width accent-bar layout (no card bubbles, no wasted horizontal space)
- [ ] User messages have a distinct accent bar color from assistant messages
- [ ] Auto-scroll to bottom on mount and on new messages (when near bottom)
- [ ] "New messages" pill appears when scrolled up and new messages arrive
- [ ] Message jump dropdown/sheet lists all messages with sequence, role, and truncated text
- [ ] Long messages (>400px rendered height) are collapsed by default with "Show more" toggle
- [ ] Search highlighting works within rendered markdown content
- [ ] All `wc-*` semantic tokens used — zero hardcoded color values in new code
- [ ] All existing `MessagesPane` tests pass (updated for new layout)
- [ ] New tests cover: MarkdownRenderer, CodeBlock, MermaidDiagram, languageDetection, MessageJumpList, auto-scroll, collapse, pill
- [ ] `npx vitest run` passes with zero failures
- [ ] No dead code, no compatibility shims, no commented-out old code
- [ ] SEAMS.md updated with new testability boundaries
