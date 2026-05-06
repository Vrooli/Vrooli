# Session Details UX Implementation Plan

## Purpose

Implement a professional session details experience for Swarm Manager agent sessions: creation actions must visibly start sessions and route to the new session, the conversation must use shared markdown-capable chat primitives, desktop must use a collapsible/resizable inspector, and mobile must use full-page tabs instead of stacking secondary content below the conversation.

This is a greenfield scenario surface. Do not preserve weak existing session-details implementations for compatibility. Replace dead or duplicated code with the target architecture in one coherent pass.

## Required Reading

Future agents must run:

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement utils-unification test react-coherence boundary-of-responsibility-enforcement
prompt-manager skill read cli-steer api-steer
```

Repository and scenario commands:

```bash
vrooli help
cd scenarios/swarm-manager && make test
```

Relevant internal docs:

- `scenarios/swarm-manager/docs/internal/AGENT-SESSIONS.md`
- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/docs/internal/COHERENCE-NOTES.md`
- `scenarios/swarm-manager/docs/internal/UTILS_UNIFICATION_NOTES.md`

## Greenfield Constraint

- Do not add compatibility wrappers around the current session details layout.
- Do not keep duplicate chat rendering code after the shared chat primitive is introduced.
- Do not introduce alternate session creation flows, alternate tab implementations, or a second drawer/resizing system.
- Do not add dependencies. Use existing Radix tabs, `useResizablePanel`, `renderMarkdown`, shared UI primitives, stores, services, and test utilities.
- Remove dead code created by the refactor in the same implementation branch.

## Problem Statement

The current session details page is functional but not product-grade:

1. `Plan Work With Agent` and `Author Operating Mode` create sessions from the graph action launcher, but the graph handler discards the returned session and does not navigate to `/sessions/:sessionId`.
2. The graph launcher closes before async create feedback is visible. It has an `error` prop, but errors are hidden unless the user reopens the menu; loading feedback is similarly absent after click.
3. `SessionDetailsPage.tsx` hand-rolls the conversation UI with plain paragraphs. It does not use the existing markdown rendering path, auto-scroll pattern, waiting indicator, or shared composer behavior used by clarification/evidence flows.
4. Desktop uses a fixed two-column grid with the inspector always consuming `320px`.
5. Mobile stacks proposals, artifacts, and details below the conversation. That makes secondary sections hard to reach and prevents each section from owning the full viewport.
6. The page owns too many responsibilities inline: session data orchestration, chat rendering, proposal rendering, artifact routing, metadata rendering, responsive layout, and mutation feedback.

## Current Technical Context

Observed files and responsibilities:

- `ui/src/surfaces/graph/components/GraphWorkspace.tsx`
  - Owns graph action launcher callbacks.
  - Calls `createSession(...)` but does not navigate to `sessionDetailPath(session.id)`.
  - Uses `isMutating` from `useAgentSessionStore` as busy state.
- `ui/src/surfaces/graph/components/GraphActionLauncher.tsx`
  - Renders the FAB menu.
  - Shows `error` only inside the menu, which is closed before async failure becomes visible.
- `ui/src/pages/SessionDetailsPage.tsx`
  - Owns the routed detail page.
  - Inlines conversation, composer, proposals, artifacts, and metadata rendering.
  - Uses `DetailPageLayout`, `DetailPageHeader`, `useAgentSessionPolling`, and `useAgentSessionStore`.
- `ui/src/components/backlog/clarification-messages.tsx`
  - Existing chat thread precedent: auto-scroll, user/assistant alignment, `renderMarkdown`, waiting spinner.
  - Domain-specific colors/types mean it should inform extraction rather than be reused as-is.
- `ui/src/components/backlog/evidence-request-messages.tsx`
  - Similar duplicated chat thread with different accent color.
- `ui/src/hooks/useResizablePanel.ts`
  - Existing performant persisted width seam used by the main sidebar.
- `ui/src/app/shell/AppShell.tsx` and `ui/src/surfaces/graph/components/sidebar/Sidebar.tsx`
  - Existing pattern for resize handle props, persisted width, and mobile/desktop conditional behavior.
- `ui/src/components/ui/tabs.tsx`
  - Existing Radix tab primitive.
- `ui/src/components/ui/drawer.tsx`
  - Responsive modal drawer, but the session inspector should be persistent/non-modal inside the page rather than portal-based.
- `ui/src/services/agent-session-service.ts`
  - Domain service seam for list/get/create/continue/refresh/cancel/apply.
- `ui/src/stores/agent-session-store.ts`
  - Store seam for session mutations and server-cache updates.
- `ui/src/app/routes/route-paths.ts`
  - `sessionDetailPath(sessionId)` already exists.

Backend/API context:

- `api/internal/agentsessions/service.go` creates durable sessions, appends the initial user message, spawns Agent Manager, records run/task IDs, and returns the loaded session.
- `api/internal/agentsessions/handler.go` exposes REST endpoints under `/api/v1/agent-sessions`.
- Existing API behavior is sufficient for the requested UX changes. No API change is planned unless validation reveals missing attachment/media access for message attachments.

## Target End State

### Session Creation

- Clicking `Plan Work With Agent` or `Author Operating Mode` closes the menu, displays a visible "Starting session..." state, creates the session once, then navigates to `/sessions/:sessionId`.
- If create fails, the graph UI shows a persistent, visible, dismissible error near the FAB or via the app's canonical local status surface. The user does not need to reopen the menu.
- Double-clicks or repeated activation while `isMutating` is true do not spawn duplicate sessions.

### Shared Chat Surface

- Session details, clarification, and evidence request flows use a shared chat primitive instead of each maintaining separate message-bubble logic.
- Messages render markdown through `renderMarkdown` and `prose-sm-slate`.
- Message role alignment, timestamps, waiting state, empty state, and auto-scroll are handled by the shared primitive.
- Composer behavior is reusable: auto-resize, Ctrl/Cmd+Enter submit, disabled/loading state, restore draft on send failure.
- Domain-specific flows can still provide accent color, attachment footer content, or extra action regions through explicit props/slots.

### Desktop Layout

- The conversation is the primary workspace and uses all available width when the inspector is collapsed.
- The inspector is a persistent right-side panel, not a page card column.
- Inspector supports:
  - Collapse/expand.
  - Resizable width using `useResizablePanel`.
  - Persisted width in localStorage, e.g. `swarm-manager.session-inspector.width.v1`.
  - Tabs or segmented section navigation for `Proposals`, `Artifacts`, and `Details`.
  - Default section selection: `Proposals` when any proposal is ready, otherwise `Artifacts` if artifacts exist, otherwise `Details`.
- The inspector resize handle follows the main sidebar's accessible separator pattern.

### Mobile Layout

- Mobile uses top-level session tabs:
  - `Conversation`
  - `Proposals`
  - `Artifacts`
  - `Details`
- Each tab owns full page content. Secondary sections are not stacked below the conversation.
- Conversation tab uses a full-height flex layout with messages scrolling and composer pinned at the bottom of the page content.
- The conversation is not wrapped in a decorative card on mobile.

## Scope

### In Scope

- Fix graph action session creation navigation and visible feedback.
- Extract shared chat primitives and migrate session details, clarification messages, and evidence request messages to them.
- Refactor `SessionDetailsPage.tsx` into smaller components/hooks.
- Implement desktop inspector collapse/resize.
- Implement mobile tab layout.
- Add focused UI tests and update affected existing tests.
- Run focused and scenario-level validation.
- Update internal docs if implementation changes architecture seams.

### Out Of Scope

- Changing Agent Sessions API semantics.
- Adding new session kinds or proposal kinds.
- Implementing global toast infrastructure unless an existing notification mechanism is found and clearly intended for this use.
- Adding attachment upload support to session composer unless the existing API/storage path already supports it end-to-end.
- Redesigning unrelated detail pages.

## Responsibility Boundaries

Use these ownership rules during implementation:

- `GraphWorkspace` owns graph-level action orchestration and route navigation after creation. It must not render session details.
- `GraphActionLauncher` owns FAB/menu presentation only. It should receive status/error props and callbacks, not call stores directly.
- `agent-session-store` owns session mutation state and cached session updates. It must not know routes.
- `agent-session-service` owns API wire calls and proto mapping. It must not know UI layout.
- `SessionDetailsPage` owns route params, top-level session loading, and page assembly only.
- `components/session/*` should own session-specific presentation: layout shell, inspector, proposal list, artifact list, metadata, and session-specific chat adapter.
- `components/chat/*` should own reusable chat presentation/composer behavior. It must not import session/backlog/review domain types.
- `lib/render-markdown.ts` remains the markdown rendering seam. Do not create a second markdown renderer.

## Implementation Strategy

### Phase 1: Lock Current Behavior With Focused Tests

Add or update tests before refactoring where practical.

1. Add `GraphWorkspace` or action-flow test coverage that stubs `createSession` and asserts:
   - clicking `Plan Work With Agent` calls `createSession` with `meta_orchestration`,
   - successful create navigates to `sessionDetailPath(session.id)`,
   - clicking `Author Operating Mode` calls `createSession` with `operating_mode_authoring`,
   - create failure renders a visible alert/status outside the closed menu,
   - busy state disables both session-start items.
2. Extend `SessionDetailsPage.test.tsx` to establish target expectations:
   - markdown in assistant messages renders as HTML, not raw markdown text,
   - mobile tabs expose Conversation/Proposals/Artifacts/Details and hide inactive tab content,
   - desktop inspector can collapse and expand,
   - proposal-ready sessions default inspector to Proposals,
   - send failure restores the composer draft and shows an alert.

### Phase 2: Fix Session Creation Navigation And Feedback

1. Import `sessionDetailPath` into `GraphWorkspace.tsx`.
2. Change `handleCreateAgentSession` to:
   - set `launcherError` to null,
   - set local visible status such as `launcherStatus = "starting"`,
   - await `const session = createSession(...)`,
   - navigate to `sessionDetailPath(session.id)`,
   - clear status after navigation or on completion.
3. Add a failure path that sets `launcherError` and clears loading.
4. Update `GraphActionLauncher` so async errors/status remain visible after the menu closes:
   - Preferred: render a compact status/alert adjacent to the FAB outside the menu.
   - Include dismiss behavior for errors.
   - Keep `role="alert"` for failures.
5. Keep duplicate-spawn prevention at the existing store `isMutating` seam and in launcher disabled state.

### Phase 3: Extract Shared Chat Primitives

Create a focused shared chat folder:

```text
ui/src/components/chat/
  ChatThread.tsx
  ChatMessageBubble.tsx
  ChatComposer.tsx
  chat-types.ts
  ChatThread.test.tsx
  ChatComposer.test.tsx
```

Proposed generic types:

```ts
export type ChatRole = "user" | "assistant" | "system";

export interface ChatMessageView {
  id: string;
  role: ChatRole;
  content: string;
  createdAt?: string;
  attachmentIds?: string[];
}
```

`ChatThread` props:

- `messages: ChatMessageView[]`
- `isWaiting?: boolean`
- `emptyLabel?: string`
- `accent?: "cyan" | "violet" | "slate"`
- `getMessageMeta?: (message) => ReactNode`
- `renderAttachmentPreview?: (message) => ReactNode`
- `testId?: string`
- `className?: string`

`ChatComposer` props:

- `value`
- `onChange`
- `onSubmit`
- `disabled`
- `isSubmitting`
- `placeholder`
- `submitLabel`
- `testId`
- optional footer/leading actions slot for future attachments.

Implementation rules:

- Use `renderMarkdown` and `dangerouslySetInnerHTML` only inside `ChatMessageBubble`.
- Keep auto-scroll in `ChatThread`.
- Keep textarea sizing in `ChatComposer` using `useAutoResizeTextarea`.
- Use existing `Textarea` and `Button`.
- Avoid a domain-specific import cycle: chat components import only shared UI/lib/hooks/types.

### Phase 4: Migrate Existing Chat Consumers And Delete Duplication

1. Update `SessionDetailsPage` to map `AgentSessionMessage` to `ChatMessageView`.
2. Update `ClarificationMessages` to become a thin adapter around `ChatThread` with cyan accent and attachment placeholders.
3. Update `EvidenceRequestMessages` to become a thin adapter around `ChatThread` with violet accent and added-evidence footer.
4. Delete duplicated bubble/scroll/spinner markup from the domain message components.
5. Ensure existing clarification/evidence tests still pass or are updated to assert behavior through the new component.

### Phase 5: Refactor Session Details Page Into Session Components

Extract session-specific components under `ui/src/components/session/`:

```text
ui/src/components/session/
  SessionConversation.tsx
  SessionInspector.tsx
  SessionInspectorSectionTabs.tsx
  SessionProposalList.tsx
  SessionArtifactList.tsx
  SessionMetadata.tsx
  session-artifact-routing.ts
  session-view-model.ts
```

Ownership:

- `SessionConversation` wraps `ChatThread` + `ChatComposer` and owns send affordance layout.
- `SessionInspector` owns desktop inspector shell, collapse control, resize handle integration, and section switching.
- `SessionProposalList`, `SessionArtifactList`, and `SessionMetadata` own only their section rendering.
- `session-artifact-routing.ts` owns `nodeIdForArtifact` / detail route mapping as a pure utility with tests.
- `session-view-model.ts` owns labels/icons/status presentation helpers if they are used by more than one component.

Delete the inline `ProposalList`, `ArtifactList`, `RunDetail`, and `nodeIdForArtifact` from `SessionDetailsPage.tsx` after migration.

### Phase 6: Implement Desktop Inspector

1. Add a `sessionDetailContainerRef` and `sessionInspectorRef` at the page/component shell level.
2. Use `useResizablePanel` with:
   - `minSize: 280`
   - `maxSize: 520`
   - `defaultSize: 340`
   - `adjacentMinSize: 480`
   - `handleWidth: 6`
   - `storageKey: "swarm-manager.session-inspector.width.v1"`
3. Render desktop shell as:
   - conversation flex child,
   - resize handle,
   - inspector flex child with explicit width.
4. Hide inspector and handle when collapsed.
5. Provide an icon button in the page/header area to reopen the inspector.
6. Ensure collapsed state persists only if that matches product expectations. Recommended: persist width, keep collapse ephemeral per page load unless a clear preference exists.

### Phase 7: Implement Mobile Tabs

1. Use existing `useIsMobile` and `Tabs`.
2. Mobile render path:
   - tab row directly under detail header,
   - `TabsContent` sections with `Conversation`, `Proposals`, `Artifacts`, `Details`.
3. Ensure only active mobile tab content is mounted or visible according to Radix behavior and test expectations.
4. Conversation tab layout:
   - full-height flex within main content,
   - messages scroll,
   - composer sticky/pinned at bottom,
   - no outer conversation card border.
5. Proposals/artifacts/details tabs render the same section components used by desktop inspector, but without nested inspector card chrome.

### Phase 8: Polish Loading, Empty, Error, And Mutation States

1. Session creation:
   - visible starting state after action click,
   - visible failure state,
   - action disabled while creating.
2. Session details:
   - page loading state remains `PageLoadingState`.
   - not-found uses `ErrorState`.
   - local mutation errors use alert with `role="alert"`.
   - send/apply/cancel/refresh loading states remain explicit.
3. Conversation:
   - empty state fits in full-height desktop/mobile layouts.
   - active/running session shows waiting indicator when the last message is from the user or status is `starting`/`running`.
4. Proposals:
   - ready proposals get clear primary action.
   - applied/failed statuses are visually distinct.
5. Artifacts:
   - non-openable artifacts are still readable with disabled affordance and no misleading external-link icon.

## Contract Decisions

- Session creation remains `POST /api/v1/agent-sessions` via `agentSessionService.create`.
- Route after create is `/sessions/:sessionId` via `sessionDetailPath`.
- No new API contract is needed for the core work.
- Message markdown is rendered client-side with existing `renderMarkdown`; API continues to store raw markdown/plain text content.
- Attachment display remains limited to existing attachment IDs unless the current API exposes retrievable attachment URLs. Do not invent a fake attachment API.
- Inspector width persistence uses localStorage through `useResizablePanel`, matching the sidebar seam.

## Testing Plan

### Unit And Component Tests

Run focused tests while implementing:

```bash
cd scenarios/swarm-manager/ui
pnpm exec vitest run src/surfaces/graph/components/GraphActionLauncher.test.tsx --minWorkers=1 --maxWorkers=1
pnpm exec vitest run src/pages/SessionDetailsPage.test.tsx --minWorkers=1 --maxWorkers=1
pnpm exec vitest run src/components/backlog/clarification-panel.test.tsx src/components/backlog/evidence-request-messages.test.tsx --minWorkers=1 --maxWorkers=1
```

Add tests for:

- `components/chat/ChatThread.test.tsx`
  - markdown rendering,
  - alignment by role,
  - waiting indicator,
  - empty state,
  - attachment/footer slot.
- `components/chat/ChatComposer.test.tsx`
  - Ctrl/Cmd+Enter submit,
  - disabled and loading states,
  - no submit on empty text,
  - value is controlled by parent.
- `components/session/session-artifact-routing.test.ts`
  - backlog refs map to backlog details,
  - initiative/capture/activity refs map correctly,
  - unsupported refs return null.
- `pages/SessionDetailsPage.test.tsx`
  - desktop inspector collapse/expand,
  - mobile tabs,
  - default inspector tab,
  - markdown message rendering,
  - send success and failure behavior,
  - proposal apply behavior remains intact,
  - artifact navigation remains intact.

### Typecheck And Lint

```bash
cd scenarios/swarm-manager/ui
pnpm run type-check
pnpm run lint
```

### Scenario Test

Use lifecycle-managed scenario test:

```bash
cd scenarios/swarm-manager
make test
```

or:

```bash
vrooli scenario test swarm-manager
```

### Manual Smoke Validation

Start via lifecycle only:

```bash
cd scenarios/swarm-manager
make start
```

Manual checks:

1. Open graph view.
2. Click `Create` -> `Plan Work With Agent`.
3. Confirm visible starting feedback appears immediately.
4. Confirm browser navigates to `/sessions/<new-id>`.
5. Confirm session appears in sidebar Sessions tab after polling/cache update.
6. Repeat for `Author Operating Mode`.
7. Force or simulate create failure and confirm visible error remains after menu closes.
8. On desktop:
   - resize inspector,
   - collapse inspector,
   - reopen inspector,
   - verify conversation width responds correctly.
9. On mobile viewport:
   - verify tabs,
   - verify conversation composer pinned at bottom,
   - verify no stacked secondary sections below conversation.
10. Send a message with markdown-like content and confirm it renders correctly when echoed/stubbed in tests or returned by fixture.

## Rollout And Validation Checklist

- [ ] Session creation navigates to new detail route.
- [ ] Session creation loading and error feedback are visible after the menu closes.
- [ ] Shared chat primitives exist and are tested.
- [ ] Session details uses shared chat primitives.
- [ ] Clarification and evidence messages use shared chat primitives.
- [ ] Duplicate chat bubble/scroll/spinner implementations are removed.
- [ ] Desktop inspector is collapsible and resizable.
- [ ] Inspector width persists through `useResizablePanel`.
- [ ] Mobile session page uses tabs instead of vertical stacking.
- [ ] Artifact routing pure utility is tested.
- [ ] No new dependencies are added.
- [ ] `pnpm run type-check` passes.
- [ ] Focused Vitest suites pass.
- [ ] `cd scenarios/swarm-manager && make test` passes, or any failures are documented with exact logs and root cause.

## Risks And Mitigations

- Risk: Shared chat extraction accidentally regresses clarification/evidence flows.
  - Mitigation: migrate with adapter components and keep their current tests passing; add focused `ChatThread` coverage.
- Risk: Page-level responsive branching duplicates too much markup.
  - Mitigation: share section components and only branch at layout shell level.
- Risk: `useResizablePanel` assumes left-to-right resizing from container left; right-side inspector may need inverse math.
  - Mitigation: either extend `useResizablePanel` with an explicit `edge: "left" | "right"` option and update sidebar tests, or wrap inspector in a container where existing math remains valid. Prefer improving the hook if the API stays simple and tests cover both directions.
- Risk: No global toast primitive exists.
  - Mitigation: use a local persistent status surface near the FAB; do not introduce a global notification system for this slice.
- Risk: Markdown rendering uses `dangerouslySetInnerHTML`.
  - Mitigation: continue using the existing `renderMarkdown` seam, which escapes HTML and is already used across the app.
- Risk: Full scenario tests may be long-running.
  - Mitigation: run focused UI tests first, then `make test` with an appropriately long timeout.

## Non-Goals And Prohibited Patterns

- Do not directly execute scenario binaries or old development scripts.
- Do not add packages.
- Do not create a second markdown renderer.
- Do not create a second tabs primitive.
- Do not create a second resize hook.
- Do not leave `SessionDetailsPage.tsx` as a large all-in-one component after the refactor.
- Do not keep old session details chat markup once `ChatThread` exists.
- Do not hide errors inside a menu that closes before the error appears.
- Do not preserve mobile stacked secondary sections.

## Definition Of Done

The work is done when:

1. Both graph action session starters create exactly one session and route to its details page.
2. Session start loading and error feedback is visible and accessible.
3. Session, clarification, and evidence conversations share the same chat rendering primitives.
4. Chat content renders markdown consistently.
5. Desktop session details has a collapsible, resizable inspector using the existing resize seam.
6. Mobile session details uses tabs with full-page tab contents.
7. The code is organized by responsibility with small components and tested pure utilities.
8. Dead duplicated chat/layout code is removed.
9. Focused UI tests, type-check, lint, and scenario validation pass or failures are documented with concrete causes.
