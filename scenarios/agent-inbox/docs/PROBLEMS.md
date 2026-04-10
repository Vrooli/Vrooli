# Known Problems & Blockers

## Open Issues

### React Error #310 - Intermittent Crash on Message Send (Investigated 2026-01-15)
**Severity:** Low (caught by ErrorBoundary, recoverable via retry)
**Symptoms:**
- React error #310 ("Too many re-renders") thrown when sending a message
- Error caught by `ChatContent` ErrorBoundary
- UI shows error fallback with retry button, does not white-screen crash

**Investigation Findings:**
- Deep audit of 20+ React files found no static code issues that would cause this
- All hooks follow proper discipline (no conditional hooks, no setState during render)
- useCompletion uses request ID tracking to prevent stale updates
- AbortController properly cancels in-flight requests
- Stable empty array constants used throughout

**Root Cause Assessment:**
- Likely an intermittent race condition during rapid state updates when:
  1. Message is sent
  2. Chat list refreshes
  3. Streaming response starts
  4. Multiple components update simultaneously
- Cannot reproduce reliably; error is transient

**Fixes Applied:**
- `TemplateSelector.tsx:255-258` - fixed inconsistent optional chaining
- `ChatView.tsx` - added granular ErrorBoundary wrappers around ChatHeader, AsyncOperationsPanel, MessageList, and MessageInput for better crash isolation (2026-01-15)

**Mitigation:**
- ErrorBoundary catches error and provides retry button
- User can retry message send, usually succeeds on second attempt
- Granular error boundaries now isolate crashes to specific sections (header, message list, or input) rather than the entire chat view

**Future Consideration:**
- If error becomes frequent, add React DevTools profiler analysis in dev mode
- Consider adding error telemetry to track frequency and stack traces

### OpenRouter Rate Limits
- Need graceful handling when rate limits are hit
- Consider request queuing or user notification

## Deferred Ideas

### Multi-user Support
- Current design is single-user
- Would need auth system and per-user data isolation
- Consider for P1/P2 iteration

### Local Model Fallback
- If OpenRouter is unavailable, could fallback to Ollama for basic chat
- Not in current scope, but worth considering

### Mobile Responsive Design
- Current design is desktop-first
- Mobile layout would require significant UI restructuring
- Explicitly out of scope per PRD

## UX Issues

### UX Audit - Primary Journey (Updated 2026-02-17)

#### High-friction areas found
- Search discoverability was low. Users had keyboard shortcuts available but no inline affordance near search.
- Empty-state onboarding was clear on desktop but hid quick usage guidance on mobile, reducing first-run clarity.
- Composer footer grouped too many controls into a single row, which could overflow or feel dense on constrained viewports.
- Key BAS/E2E selectors in high-traffic surfaces were hardcoded in components instead of consistently sourced from the selector registry.

#### Improvements applied
- Added inline search affordance hint in sidebar (`/` and `Ctrl+K`) and maintained quick/content search path visibility.
- Kept quick tips visible on mobile empty state with compact spacing to preserve first-time guidance.
- Reduced composer friction by splitting keyboard/help guidance from status chips and allowing wrapped indicator layout on mobile.
- Added explicit slash-command affordance (`/` for tools) near the composer.
- Moved key primary-flow test selectors to the selector registry (`app`, `chatListPanel`, `messageInput`, `emptyState`) and used registry references in App/Sidebar/EmptyState/MessageInput.
- Collapsed mobile sidebar secondary actions (labels, settings, shortcuts, multi-select mode) into a single overflow menu to reduce header/footer clutter.
- Condensed mobile chat-header action density by moving read/star/archive actions into the existing "more actions" menu while preserving direct desktop affordances.
- Added selector-registry coverage for new mobile overflow controls (`sidebar.mobileActionsButton`, `chatHeader.mobileActionsButton`) and used registry lookups in the updated components.
- Reduced mobile chat-header duplication by hiding the in-chat title/rename row at small breakpoints and keeping model/tools in a compact horizontal row.
- Tightened chat mode/status chrome on mobile by reducing spacing and forcing overflow containment instead of wrapped multi-line controls.
- Streamlined mobile composer ergonomics: smaller input shell spacing, safe-area bottom padding, smaller send button, and removal of non-essential keyboard helper text on phone viewports.
- Removed mobile-only suggestion chips from the always-visible composer footer to prevent large, scroll-heavy stacks below the input.
- Simplified top mobile app bar by removing duplicated star action (still available in chat actions menu), reducing header clutter and preserving vertical space.
- Moved mobile chat actions into the top app header (labels, read/star/archive, rename, export, delete) and hid the in-chat desktop header on mobile.
- Added editable mobile chat title parity by wiring rename from the top header with the same underlying update behavior.
- Fixed message bubble overflow on constrained viewports by adding explicit flex shrink containment (`min-w-0`) and markdown/link word-wrapping (`overflow-wrap:anywhere`, `break-all`) so long tokens cannot render off-screen.
- Extended overflow containment to agent-mode rendering surfaces (`AgentEventList`, `AgentMessageBubble`, inline code) so long content in agent chats also wraps/clamps correctly on mobile.
- Removed agent-only typography (`prose`) wrappers and added global `markdown-content` containment rules so fenced code blocks/ASCII diagrams render inside bubbles with internal horizontal scroll instead of clipping off-screen.

#### Remaining UX debt
- Content-search advanced toggles (`Aa`, `W`, `.*`) still depend on tooltip learning; an inline legend or settings sheet would improve first-pass comprehension.
- Long-list chat management can still benefit from stronger view-specific empty-state CTAs for inbox/starred/archived.
- Bulk-operation toolbar is compact but icon-only; optional text labels on first use may improve learnability without increasing steady-state clutter.
- Mode switching is still rendered in a dedicated row; moving mode controls directly into composer/header actions could recover additional vertical space on small screens.

## Resolved

### PostgreSQL Dependency Blocking Deployment (Resolved 2026-02-12)
**Issue:** PostgreSQL required a running external service, blocking standalone desktop apps, mobile apps, and LPBS app store bundling.
**Resolution:** Greenfield rewrite from PostgreSQL to embedded SQLite. Pure-Go driver (modernc.org/sqlite), FTS5 for search, WAL mode, zero external dependencies. All tests pass with in-memory SQLite.
