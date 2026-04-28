# Swarm Manager UI Route Hard Cutover Implementation Plan

Last updated: 2026-04-28

## Purpose

Replace Swarm Manager's current fullscreen-overlay navigation model with a first-class route architecture. Fullscreen detail surfaces, Command Post, and Decision Stream must become real React Router pages so browser history, in-app back controls, direct URLs, and keyboard navigation all describe the same user journey.

This is an implementation handoff plan. No code has been changed for this plan.

## Required Reading

Run these first:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
sed -n '1,260p' scenarios/prompt-manager/store/skills/packs/core/implementation-plan-authoring/SKILL.md
sed -n '1,220p' scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md
```

Then inspect the current routing evidence:

```bash
sed -n '1,130p' scenarios/swarm-manager/ui/src/App.tsx
sed -n '1,190p' scenarios/swarm-manager/ui/src/hooks/useDetailUrlSync.ts
sed -n '1,170p' scenarios/swarm-manager/ui/src/hooks/useDetailNavigation.ts
sed -n '70,335p' scenarios/swarm-manager/ui/src/surfaces/graph/components/GraphWorkspace.tsx
sed -n '1,180p' scenarios/swarm-manager/ui/src/components/command-post/CommandPostOverlay.tsx
sed -n '55,115p' scenarios/swarm-manager/ui/src/components/detail/DetailPageHeader.tsx
```

## Greenfield Hard-Cutover Rule

This work is a hard cutover. Do not add compatibility layers, redirect bridges, deprecation aliases, dual routing, URL migration helpers, legacy query parsing, or dead transitional code.

Required removals:

- Remove `LegacyRedirect` routes and tests.
- Remove `?detail=...` as an opening mechanism.
- Remove `/details/...` route handling entirely.
- Remove `useDetailUrlSync`.
- Remove `useDetailNavigation` as the central detail-selection navigation API.
- Remove `detail-selection-store` if no remaining production code needs it after route-param conversion.
- Remove Command Post fullscreen overlay state from `GraphWorkspace`.

If an existing test asserts old behavior, rewrite it to assert the new route contract or delete it only when the covered behavior no longer exists.

## Problem Statement

Swarm Manager's UI currently renders many fullscreen experiences as overlays under `/graph`. They look like pages but behave like dialogs. The result is confusing navigation:

- A detail page header left arrow calls `closeDetail()` instead of browser/application back.
- Details are stored in Zustand and mirrored to query params using `replace`, so opening details does not build a real browser history trail.
- Command Post and Decision Stream are local React state under `GraphWorkspace`, not routable screens.
- The Command Post summary has an `X`, reinforcing dialog semantics despite being fullscreen.
- Browser Back cannot reliably traverse `graph -> command post -> decision stream -> detail -> previous page`.
- Deep links are query overlays rather than canonical URLs.

The operator mental model in `scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md` already says the graph workspace is primary with "detail routes". The implementation does not yet match that model.

## Scope

In scope:

- React Router route table replacement for graph, detail, Command Post, and Decision Stream pages.
- Route utilities for canonical path building/parsing.
- Route-aware page shell/header behavior.
- Page components reading route params instead of global detail-selection state.
- Graph/sidebar/inspector/command-post navigation callers using `navigate(...)`.
- Keyboard shortcut behavior updated to route semantics.
- Tests for route utilities, route rendering, back/fallback behavior, and old-pattern removal.
- Documentation updates for the UI architecture and route contract.

Out of scope:

- API endpoint changes.
- CLI changes.
- Business logic changes in backlog, scenario, execution, initiative, capture, review, or workshop flows.
- Visual redesign beyond header semantics needed by route behavior.
- Adding backward-compatible support for old paths or query detail URLs.

## Current Technical Context

Current route table:

- `ui/src/App.tsx` has one primary route: `/graph`.
- `/backlog`, `/backlog/:kind/:name`, `/scenarios`, `/scenarios/:name`, `/execution`, `/prompts`, `/settings`, and `/details/...` are redirects into `/graph` or `/graph?detail=...`.

Current fullscreen state:

- `ui/src/surfaces/graph/components/GraphWorkspace.tsx` owns `showCommandPost` local state.
- `GraphWorkspace` calls `useDetailUrlSync()` and reads `detailSelection`.
- Detail pages render inside `GraphWorkspace` as `absolute inset-0 z-40` overlays.
- Command Post renders inside `GraphWorkspace` as `absolute inset-0 z-[60]`.

Current detail selection:

- `ui/src/stores/detail-selection-store.ts` stores one mutually exclusive detail selection.
- `ui/src/hooks/useDetailUrlSync.ts` mirrors this state to query params with `{ replace: true }`.
- `ui/src/hooks/useDetailNavigation.ts` mutates the selection store and sidebar state.
- `ui/src/components/detail/DetailPageHeader.tsx` calls `closeDetail()` on desktop and `openSidebar()` on mobile.

Current Command Post:

- `ui/src/components/command-post/CommandPostOverlay.tsx` manages `summary` vs `decision-stream` with local component state.
- Escape returns from decision stream to summary or closes the overlay.
- The summary header has an `X` close button.

Current tests affected:

- `ui/src/hooks/useDetailNavigation.test.tsx`
- `ui/src/hooks/use-url-state.test.tsx`
- `ui/src/components/detail/DetailPageHeader.test.tsx`
- `ui/src/components/command-post/CommandPostOverlay.test.tsx`
- `ui/src/surfaces/graph/components/LegacyRedirect.test.tsx`
- Page tests using `MemoryRouter initialEntries={["/graph"]}` plus `detail-selection-store` setup.

## Target End State

Canonical routes:

```text
/                         -> index route to /graph
/graph                    -> graph workspace, default topology lens
/graph/focus              -> graph workspace, focus lens
/graph/topology           -> graph workspace, topology lens
/graph/operations         -> graph workspace, operations lens
/backlog/:kind/:name      -> backlog detail page
/scenarios/:name          -> scenario detail page
/executions/:executionId  -> execution detail page
/initiatives/:name        -> initiative detail page
/captures/:captureId      -> capture detail page
/command-post             -> Command Post summary page
/command-post/decisions   -> Decision Stream page
/not-found or *           -> NotFoundPage
```

Query params that remain valid:

- Graph state: `focus`, `returnLens`, `select`, sidebar/search/filter params.
- Detail substate: `tab`, `file`, `items`, and other page-local view params.
- Command Post/Decision Stream local filters only if they are share-worthy. Otherwise keep local state within that route component.

Query params that must be removed:

- `detail`
- `kind` as a route discriminator for detail pages
- `name` as a generic detail identifier
- `execId`
- `id` for capture detail selection

Back behavior:

- Detail page header left arrow calls route-aware back behavior:
  - If there is in-app history, `navigate(-1)`.
  - If direct loaded, navigate to a route-specific fallback, usually `/graph/topology`.
- Command Post summary header uses a left arrow, not `X`.
- Decision Stream left arrow navigates to `/command-post`.
- Escape follows the same route semantics:
  - On a detail route, back/fallback.
  - On `/command-post/decisions`, navigate to `/command-post`.
  - On `/command-post`, back/fallback to `/graph/topology`.
- Mobile detail header must still expose sidebar access where needed, but it cannot replace route back. Use a separate menu/sidebar control if needed.

Architecture shape:

```text
ui/src/
  app/
    App.tsx or routes/AppRoutes.tsx
    routes/
      route-paths.ts       # canonical path constants/builders/parsers
      route-contracts.ts   # route param validators + fallback policy
      useAppBack.ts        # route-aware back/fallback behavior
      AppRoutes.test.tsx
      route-paths.test.ts
  pages/
    graph/
      GraphWorkspacePage.tsx
    detail/
      BacklogDetailRoute.tsx
      ScenarioDetailRoute.tsx
      ExecutionDetailRoute.tsx
      InitiativeDetailRoute.tsx
      CaptureDetailRoute.tsx
    command-post/
      CommandPostPage.tsx
      DecisionStreamPage.tsx
  surfaces/
    graph/                 # graph canvas/sidebar/HUD only, no detail page ownership
  components/
    detail/                # reusable detail page shell/header/lens bar
    command-post/          # route page content components, not overlay ownership
```

Adapt paths to existing conventions, but preserve these boundaries:

- `app/routes` owns route names, builders, validators, and back fallback rules.
- `pages/*Route.tsx` adapts route params into page props/data hooks.
- `surfaces/graph` owns graph exploration only.
- `components/detail` owns shared detail chrome, not navigation state.
- Domain detail components receive explicit identifiers from route adapters, not global selection.

## Implementation Strategy

### Phase 1: Route Contract Foundation

Deliverables:

- Create canonical route utility module.
- Add typed builders:
  - `graphPath({ lens?, focus?, returnLens?, select? })`
  - `backlogDetailPath(kind, name, query?)`
  - `scenarioDetailPath(name, query?)`
  - `executionDetailPath(executionId, query?)`
  - `initiativeDetailPath(name, query?)`
  - `captureDetailPath(captureId, query?)`
  - `commandPostPath(query?)`
  - `decisionStreamPath(query?)`
- Add param validators for backlog kind, graph lens, and required entity identifiers.
- Add `useAppBack({ fallback })` with a testable seam around history depth.

Testing:

- Unit-test path encoding for names with spaces/slashes/special chars.
- Unit-test fallback decisions for direct-load vs in-app history.
- Unit-test graph lens validation.

### Phase 2: Replace App Route Table

Deliverables:

- Replace `App.tsx` routes with only canonical routes.
- Remove `LegacyRedirect.tsx` and `LegacyRedirect.test.tsx`.
- Make root index navigate to `/graph`.
- Keep `getRouterBasename()` behavior intact because proxy basename remains valid.
- Add `AppRoutes.test.tsx` to verify each canonical path renders the expected route shell.

Testing:

- `MemoryRouter` route tests for all canonical routes.
- A negative test that old paths (`/details/...`, `/graph?detail=...`) do not render detail pages.

### Phase 3: Detach GraphWorkspace From Page Overlays

Deliverables:

- Remove detail page lazy imports from `GraphWorkspace`.
- Remove `detailSelection` rendering block.
- Remove `useDetailUrlSync()`.
- Remove `showCommandPost` state and `CommandPostOverlay` from `GraphWorkspace`.
- Update graph HUD/sidebar Command Post buttons to `navigate(commandPostPath())`.
- Update sidebar item clicks and graph inspector "Open Details" to `navigate(detailPathFromNode(...))`.
- Keep graph node selection (`select`) distinct from page navigation; clicking list rows can either select in graph or open details based on current intended UX, but the detail opening path must be route navigation.

Testing:

- Graph workspace test confirms it does not render `detail-overlay` or `command-post-overlay`.
- Inspector/Sidebar interaction tests assert `navigate(...)` with canonical paths.

### Phase 4: Convert Detail Pages To Route Params

Deliverables:

- Each detail route adapter reads `useParams()` and validates params.
- Refactor page components to accept explicit identifiers:
  - Backlog: `{ kind, name }`
  - Scenario: `{ name }`
  - Execution: `{ executionId }`
  - Initiative: `{ name }`
  - Capture: `{ captureId }`
- Keep subview state in query params via `useUrlState`, but ensure it is scoped to the current route.
- Replace `closeDetail` dependencies with `useAppBack({ fallback: graphPath({ lens: "topology" }) })` or a domain-specific fallback.
- Delete `selectionToNodeId` dependency by introducing route-param based node-id builders in a route/domain utility.
- Ensure capture details are part of the official detail route registry if the registry remains.

Testing:

- Page tests render via canonical routes, not `/graph` plus store setup.
- Header click tests assert navigation/back behavior.
- Direct-load fallback tests for each detail route.
- Invalid param tests render `NotFoundPage` or a route-local invalid state with a route fallback.

### Phase 5: Route Command Post And Decision Stream

Deliverables:

- Replace `CommandPostOverlay` with route page components:
  - `CommandPostPage`: owns summary query and summary layout.
  - `DecisionStreamPage`: owns decision stream query/state and renders `DecisionStreamView`.
- `SummaryView` enters decisions via `navigate(decisionStreamPath())`.
- `DecisionStreamView` back action navigates to `commandPostPath()`.
- Opening an item from Command Post navigates to the canonical detail route.
- Summary header uses shared page header/back affordance, not `X`.
- Escape behavior delegates to route navigation.
- Keep true dialogs inside Command Post, such as run modals and clarification panel, as dialogs/panels.

Testing:

- Command Post route renders summary at `/command-post`.
- Decision Stream route renders at `/command-post/decisions`.
- Entering decisions pushes a history entry.
- Back from decisions returns to summary.
- Opening a backlog/execution item navigates to canonical detail route.
- Old overlay-specific tests are removed or rewritten as page-route tests.

### Phase 6: Keyboard, Mobile, And Header Semantics

Deliverables:

- Update `DetailPageHeader` props so navigation is injected:
  - `onBack`
  - `backLabel`
  - optional `onOpenSidebar` if a route genuinely needs mobile sidebar access.
- Desktop and mobile both expose route back. Mobile sidebar access must be a separate button where required.
- Update `useGraphKeyboardShortcuts`:
  - Remove direct `detail-selection-store` dependency.
  - Escape on graph still deselects nodes.
  - `P` navigates to Command Post instead of toggling local overlay.
  - Number keys continue switching graph lenses via route/path updates.
- Add route keyboard handler for Command Post/detail pages if Escape is expected globally outside graph.

Testing:

- Header tests assert left arrow calls `navigate(-1)` or fallback seam, not store clearing.
- Mobile tests assert back remains available and sidebar access is separate if retained.
- Keyboard shortcut tests assert no detail-selection store use.

### Phase 7: Delete Obsolete State And Utilities

Deliverables:

- Delete:
  - `useDetailNavigation.ts`
  - `useDetailNavigation.test.tsx`
  - `useDetailUrlSync.ts`
  - `LegacyRedirect.tsx`
  - `LegacyRedirect.test.tsx`
  - `CommandPostOverlay.tsx` if fully replaced
  - `CommandPostOverlay.test.tsx` if fully replaced
  - `detail-selection-store.ts` if no production callers remain
- Remove exports from barrel files.
- Remove stale selectors tied only to deleted overlays, or rename selectors to page terminology.
- Update architecture docs to say detail pages and Command Post are first-class routes.

Validation commands:

```bash
rg "useDetailNavigation|useDetailUrlSync|detail-selection-store|LegacyRedirect|CommandPostOverlay|detail-overlay|command-post-overlay|detail=" scenarios/swarm-manager/ui/src
rg "details/" scenarios/swarm-manager/ui/src scenarios/swarm-manager/docs
```

Expected result: no production references, except plan/docs describing removed behavior.

### Phase 8: Full Validation

Run from `scenarios/swarm-manager`:

```bash
make test
```

Run UI-focused gates directly if scenario test output does not clearly include them:

```bash
cd scenarios/swarm-manager/ui
pnpm run type-check
pnpm run lint
pnpm run test
pnpm run build
```

Given UI build can take 5-10 minutes, set command timeouts accordingly.

If using browser-level validation, use lifecycle-managed startup only:

```bash
cd scenarios/swarm-manager
make start
make logs
make stop
```

Do not run scenario binaries directly.

## Contract Decisions

### URL Contract

Canonical page identity is path-based, not query-based.

- Good: `/backlog/execute/add-route-system?tab=files&file=plan.md`
- Bad: `/graph?detail=backlog&kind=execute&name=add-route-system&tab=files`

Entity identifiers must be encoded with route builders. Call sites must not hand-concatenate URLs.

### History Contract

Opening fullscreen pages must push browser history. Page-local view refinements may replace history.

- Push: graph to detail, graph to Command Post, Command Post to Decision Stream, Command Post to detail, detail to another detail.
- Replace: tab changes, selected file changes, graph pan/selection refinements where existing UX expects low history noise.

### Back Contract

All fullscreen page headers use route-aware back:

- Prefer `navigate(-1)` when entered from another app route.
- Fall back to canonical graph route on direct load.
- Never clear hidden global detail state as a substitute for navigation.

### Graph Contract

Graph is a workspace, not a parent overlay host for every page.

- It owns canvas, HUD, sidebar, inspector, settings drawer, stats panel, help panel, and capture compose panel.
- It does not own detail page rendering.
- It does not own Command Post rendering.

### Dialog Contract

Use dialogs/drawers/panels only for transient, interruptive, or subordinate tasks:

- Confirm delete dialogs.
- Follow-up sheets.
- Run modals.
- Settings drawer.
- Capture compose panel.
- Media lightbox.

Fullscreen workspaces are routes.

## Testing Plan

Unit tests:

- `route-paths.test.ts`: builders, encoders, validators.
- `useAppBack.test.tsx`: history/fallback behavior with injected seam.
- `DetailPageHeader.test.tsx`: route back and optional sidebar control.
- `use-url-state.test.tsx`: continue proving `replace` behavior for substate.

Route integration tests:

- `AppRoutes.test.tsx`: canonical route rendering.
- Detail route tests for each entity.
- Command Post and Decision Stream route tests.
- Graph route tests for lens path behavior.

Component rewrite tests:

- Rewrite page tests to use route params instead of `detail-selection-store`.
- Rewrite NodeInspector/Sidebar tests to assert canonical navigation paths.
- Rewrite Command Post tests around `CommandPostPage` and `DecisionStreamPage`.

Regression grep tests:

- Add a small test or script-backed assertion that production source has no imports of deleted routing primitives.
- At minimum run the `rg` validation commands from Phase 7 and record clean output in the final implementation note.

Browser smoke:

- Add BAS workflows under `scenarios/swarm-manager/bas/` only if the project expects browser automation for UI route coverage in this item.
- Suggested flows:
  - Load `/graph/topology`, open Command Post, enter Decision Stream, back to Command Post, back to graph.
  - Load `/backlog/:kind/:name` directly, use back fallback to graph.
  - From Command Post, open a backlog item and use browser Back to return to Command Post.
- After adding BAS workflows, run:

```bash
test-genie registry build
```

## Rollout / Validation Checklist

- [ ] `App.tsx` contains only canonical route definitions.
- [ ] Detail pages are not rendered from `GraphWorkspace`.
- [ ] Command Post is not rendered from `GraphWorkspace`.
- [ ] No production import of `useDetailNavigation`.
- [ ] No production import of `useDetailUrlSync`.
- [ ] No production import of `detail-selection-store`.
- [ ] No production import of `LegacyRedirect`.
- [ ] No production import of `CommandPostOverlay`.
- [ ] No production URL builder emits `?detail=`.
- [ ] Detail headers use route-aware back.
- [ ] Command Post summary uses a left-arrow route back affordance.
- [ ] Decision Stream is reachable at `/command-post/decisions`.
- [ ] Browser Back from Decision Stream returns to Command Post when entered from Command Post.
- [ ] Browser Back from a Command Post-opened detail returns to Command Post.
- [ ] Direct-loaded details fall back to `/graph/topology`.
- [ ] `pnpm run type-check` passes.
- [ ] `pnpm run lint` passes.
- [ ] `pnpm run test` passes.
- [ ] `pnpm run build` passes.
- [ ] `make test` passes or any failure is documented as unrelated with evidence.

## Risks And Mitigations

Risk: Direct-load back detection is brittle in MemoryRouter and browser history.

- Mitigation: Build `useAppBack` around an injectable history-depth/navigation seam. Unit-test the seam and route behavior separately.

Risk: Page components currently assume `detail-selection-store`.

- Mitigation: Introduce route adapter components first, then convert one detail page at a time to explicit props. Do not preserve store fallback.

Risk: Graph and detail share query param names such as `tab`, `file`, `select`.

- Mitigation: Keep page-local params on page routes. Keep graph params on graph routes. Route utilities must document ownership.

Risk: Capture support is inconsistent.

- Mitigation: Treat capture details as first-class `/captures/:captureId` routes. If the graph inspector should open captures, add capture to the official detail registry during the hard cutover.

Risk: Tests may pass while old paths still exist.

- Mitigation: Add grep-backed validation for banned symbols and banned URL patterns.

Risk: Mobile header currently replaces back with sidebar open.

- Mitigation: Make back non-negotiable on route pages. Add a separate sidebar/menu control only where route pages truly need it.

## Non-goals / Prohibited Patterns

Do not:

- Keep `/details/...`.
- Keep `/graph?detail=...`.
- Add redirects from old detail URLs.
- Add a migration layer from query-detail URLs to route-detail URLs.
- Keep duplicate route builders.
- Hand-build entity detail URLs in components.
- Keep global detail selection as a parallel page identity source.
- Keep fullscreen pages as graph overlays.
- Use `replace: true` for opening fullscreen pages.
- Hide obsolete route behavior behind comments or unused exports.

## Definition Of Done

This work is complete only when:

1. Fullscreen details, Command Post, and Decision Stream are canonical routes.
2. Browser Back and in-app back controls traverse the actual user journey.
3. Direct-loaded fullscreen routes have deterministic fallback behavior.
4. GraphWorkspace no longer owns detail or Command Post page overlays.
5. Old query-detail and legacy detail route mechanisms are fully removed, not bridged.
6. Route builders and back behavior are centralized, typed, documented, and tested.
7. Page components use route params or explicit props, not global detail-selection state.
8. Command Post and Decision Stream tests cover route transitions.
9. Grep validation proves banned old routing primitives are absent from production source.
10. `make test`, UI type-check, lint, test, and build gates pass or have documented unrelated blockers with command evidence.
