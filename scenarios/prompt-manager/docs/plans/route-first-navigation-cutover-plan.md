# Prompt Manager Route-First Navigation Cutover Plan

## Purpose

Replace prompt-manager's current selection-as-dialog navigation with the route-first navigation model recently adopted by swarm-manager. Entity detail views must become first-class pages that push browser history, support direct links, and close/back using route semantics.

## Required Reading

Future implementers must start with:

```bash
prompt-manager skill read implementation-plan-authoring react-coherence react-stability utils-unification
prompt-manager skill read cli-steer api-steer seam-discovery-and-enforcement visited-tracker-tools
```

Primary comparison files:

```bash
sed -n '1,220p' scenarios/swarm-manager/ui/src/App.tsx
sed -n '1,260p' scenarios/swarm-manager/ui/src/app/routes/route-paths.ts
sed -n '1,220p' scenarios/swarm-manager/ui/src/app/routes/useAppBack.ts
sed -n '1,220p' scenarios/swarm-manager/ui/src/app/routes/useEscapeRouteBack.ts
sed -n '1,260p' scenarios/swarm-manager/ui/src/app/shell/AppShell.tsx
sed -n '1,220p' scenarios/swarm-manager/ui/src/components/detail/DetailPageHeader.tsx
sed -n '1,340p' scenarios/prompt-manager/ui/src/hooks/useUrlState.ts
sed -n '1,760p' scenarios/prompt-manager/ui/src/components/layout/SkillManagerLayout.tsx
sed -n '1220,1810p' scenarios/prompt-manager/ui/src/components/layout/SkillManagerLayout.tsx
```

## Hard Rule: Greenfield Cutover

This is a hard cutover, not a compatibility migration.

- Do not keep `useUrlState` as a bridge.
- Do not keep query routes such as `/?skill=...`, `/?agent=...`, or `/?view=graph`.
- Do not leave alternate navigation helpers that mutate selection state instead of routing.
- Do not implement redirect compatibility from old query URLs unless the user explicitly changes the requirement.
- Delete dead tests and stale comments that describe the old query-state contract.

## Problem Statement

Prompt-manager currently treats detail views as state branches inside `SkillManagerLayout`. Selecting a skill, agent, team, run, topic, or action mutates Zustand selection state. `useUrlState` then mirrors that state into query parameters via `window.history.replaceState`, so opening details does not push a browser history entry. Closing an editor clears selection and returns to the world or graph view. In practice this makes Back/Forward unreliable and makes detail views feel like transient dialogs rather than pages.

Swarm-manager solved the same UX issue by using `react-router-dom`, a persistent `AppShell`, first-class routes, route-path helpers, and `useAppBack` fallback semantics. Prompt-manager should adopt that model directly.

## Scope

In scope:

- Add route-first app structure to prompt-manager UI.
- Promote world, graph, and all entity detail editors to real routes.
- Move URL parsing/building into route helper utilities.
- Convert sidebar/home/entity selection callbacks to `navigate(...)`.
- Replace editor close buttons and Escape behavior with route back semantics.
- Preserve existing editor behavior, dirty-state storage, mobile sidebar behavior, and deep-linkable tab/highlight state through route params/search params.
- Add tests covering route path helpers, back behavior, route rendering, and deletion of old query-state behavior.

Out of scope:

- API or CLI changes.
- Visual redesign beyond header/navigation affordances needed for route-first UX.
- Old URL compatibility.
- New product features unrelated to navigation.

## Current Technical Context

Swarm-manager target pattern:

- `scenarios/swarm-manager/ui/src/App.tsx` wraps the app in `BrowserRouter`, defines real routes, lazy-loads pages, computes basename via `getProxyInfo`, and nests pages under `AppShell`.
- `scenarios/swarm-manager/ui/src/app/shell/AppShell.tsx` owns persistent sidebar, polling, shared settings drawer, mobile sidebar collapse, and routes page content through `<Outlet />`.
- `scenarios/swarm-manager/ui/src/app/routes/route-paths.ts` centralizes canonical path builders and node/entity conversions.
- `scenarios/swarm-manager/ui/src/app/routes/useAppBack.ts` uses `window.history.state.idx` to go back when in-app history exists, otherwise replaces with a graph fallback.
- `scenarios/swarm-manager/ui/src/components/detail/DetailPageHeader.tsx` exposes a sidebar button and a close button whose close path is route-backed.
- `scenarios/swarm-manager/ui/src/pages/*DetailsPage.tsx` read entity IDs from `useParams` and tab state from `useSearchParams` or the small `use-url-state` query hook.

Prompt-manager current pattern:

- `scenarios/prompt-manager/ui/src/App.tsx` renders `SkillManagerLayout` directly. There is no router.
- `scenarios/prompt-manager/ui/package.json` does not list `react-router-dom`; swarm-manager uses `react-router-dom@^6.28.0`.
- `scenarios/prompt-manager/ui/src/hooks/useUrlState.ts` owns query params and manually calls `window.history.replaceState`.
- `scenarios/prompt-manager/ui/src/components/layout/SkillManagerLayout.tsx` owns data loading, sidebar, entity selection state, URL sync, mobile sidebar drawer, dialogs, and all editor rendering.
- `scenarios/prompt-manager/ui/src/stores/selectionStore.ts` uses selected entity IDs as navigation state and persists `pm.viewMode` for world/graph.
- `SkillEditorPanel` renders `WorldCanvas` or `GraphView` when no skill is selected, making home surfaces an editor empty state rather than pages.
- Entity editor panels already accept `onClose` or have internal close behavior; these should be rewired to route back.

## Target End State

Prompt-manager should have the same navigation architecture class as swarm-manager:

- `App.tsx` creates `BrowserRouter` with proxy basename support.
- `app/routes/route-paths.ts` is the only canonical route construction module.
- `app/routes/useAppBack.ts` and `app/routes/useEscapeRouteBack.ts` provide route back semantics.
- `app/shell/AppShell.tsx` owns the persistent sidebar, mobile drawer, resize state, polling, settings dialog, dirty menu wiring, and `<Outlet />`.
- Top-level pages/surfaces render content by route:
  - `/world`
  - `/graph`
  - `/skills/:skillId`
  - `/agents/:agentId`
  - `/teams/:teamId`
  - `/runs/:runId`
  - `/topics/:topicId`
  - `/actions/:actionId`
  - `/topics/new` or `/topics/wizard` if the wizard should remain a routable full-page flow
  - `/` redirects to the preferred home route, initially `/world` unless product owners decide graph should be default.
- Entity tab/highlight state remains in search params on the owning route:
  - `/teams/:teamId?tab=activity&subTab=decisions`
  - `/skills/:skillId?hlFile=...&hlLine=...&hlText=...`
  - `/agents/:agentId?tab=prompt&hlFile=...`
- Sidebar actions call `navigate(routePath(...))`.
- Close buttons call `useAppBack`, falling back to the last home route or `/world`.
- Browser Back from an entity detail route returns to the prior route, including `/world` or `/graph`.

## Implementation Strategy

### Phase 1: Dependency and route contract

1. Add `react-router-dom` to prompt-manager UI dependencies using the same major line as swarm-manager.
2. Create `scenarios/prompt-manager/ui/src/app/routes/route-paths.ts`.
3. Define route target types for `skill`, `agent`, `team`, `run`, `topic`, `action`, and home view routes.
4. Implement `worldPath`, `graphPath`, `skillDetailPath`, `agentDetailPath`, `teamDetailPath`, `runDetailPath`, `topicDetailPath`, `actionDetailPath`, `topicWizardPath`, and a generic `detailPath`.
5. Add `route-paths.test.ts` matching swarm-manager's coverage style.

Acceptance:

- Every route builder URL-encodes IDs.
- Query params are omitted when null/empty.
- No component builds detail URLs by hand.

### Phase 2: App router and shell skeleton

1. Replace direct `SkillManagerLayout` rendering in `App.tsx` with `BrowserRouter`, `Routes`, `Route`, `Navigate`, lazy page imports, and proxy basename logic copied in spirit from swarm-manager.
2. Create `app/shell/AppShell.tsx` and move the persistent layout responsibilities out of `SkillManagerLayout`.
3. Create an `AppShellContext` for `openSidebar`, `closeSidebar`, and `toggleSidebar` so page headers can open the sidebar on mobile/compact layouts.
4. Keep `Toaster`, `ThemeProvider`, and existing error boundaries intact.
5. Add a `NotFoundPage` or small in-place 404 route.

Acceptance:

- App renders under a real `BrowserRouter`.
- The sidebar persists while route content changes through `<Outlet />`.
- `/` redirects with `replace` to `/world`.

### Phase 3: Split layout into shell-owned state and route pages

1. Decompose `SkillManagerLayout` instead of extending it.
2. Move sidebar/data/polling/shared handlers into `AppShell`.
3. Create route pages that each own only the editor rendering for one route:
   - `pages/WorldPage.tsx`
   - `pages/GraphPage.tsx`
   - `pages/SkillDetailsPage.tsx`
   - `pages/AgentDetailsPage.tsx`
   - `pages/TeamDetailsPage.tsx`
   - `pages/RunDetailsPage.tsx`
   - `pages/TopicDetailsPage.tsx`
   - `pages/ActionDetailsPage.tsx`
   - optional `pages/TopicWizardPage.tsx`
4. Page components should read IDs with `useParams` and search state with `useSearchParams`, not from navigation selection state.
5. Keep editor stores for form data and dirty tracking; remove their role as the router.
6. Provide shared data/editor context from the shell only where it avoids duplicated fetches. Prefer a focused `PromptManagerShellContext` over a catch-all global.

Acceptance:

- No route page decides its identity from `selectedSkillId`, `selectedAgentId`, etc.
- World and graph are pages, not `SkillEditorPanel` empty states.
- Editor components remain mostly presentational and receive `onClose={goBack}`.

### Phase 4: Navigation callback conversion

1. Replace every sidebar `setSelected*` navigation callback with `navigate(...)`.
2. Convert creation flows to navigate to the newly created route after API success.
3. Convert duplicate/copy flows to navigate to the copied entity route.
4. Convert pending-decision and running-agent callbacks:
   - running member: `teamDetailPath(teamId, { tab: "members", member: agentId })` or the chosen explicit query contract
   - decision log: `teamDetailPath(teamId, { tab: "activity", subTab: "decisions" })`
5. Convert cross-reference navigation to route paths with highlight query params.
6. Convert home button to `navigate(worldPath())` or `navigate(graphPath())` based on the clicked control's intent. Do not clear selection to simulate home.

Acceptance:

- Opening any entity from sidebar, graph, world, search, pending decision popover, or xref pushes a history entry.
- No navigation callback relies on clearing another selected entity ID.

### Phase 5: Detail header and close semantics

1. Add prompt-manager equivalents of swarm-manager's `DetailPageHeader`, `DetailPageLayout`, `useAppBack`, and `useEscapeRouteBack`, adapted to existing visual components.
2. Update entity editor panels to use route close/back:
   - `SkillEditorPanel`
   - `AgentEditorPanel`
   - `TeamEditorPanel`
   - `RunEditorPanel`
   - `TopicEditorPanel`
   - `ActionEditorPanel`
3. Remove direct `useSelectionStore` close behavior from `SkillEditorPanel`.
4. Ensure Escape closes dialogs first, then uses route back for page close.
5. Preserve mobile sidebar button semantics: menu opens sidebar; close button closes page.

Acceptance:

- Header close button performs browser back when in-app history exists.
- Direct-loaded detail URLs close with `replace` to `/world` or the configured fallback.
- Mobile header menu never masquerades as route close.

### Phase 6: Delete old URL and selection navigation code

1. Delete `scenarios/prompt-manager/ui/src/hooks/useUrlState.ts` and `useUrlState.test.ts`.
2. Remove all `updateUrl` effects from `SkillManagerLayout` during decomposition.
3. Remove navigation fields from `selectionStore` that only exist to choose detail pages:
   - `selectedSkillId`
   - `selectedAgentId`
   - `selectedTeamId`
   - `selectedRunId`
   - `selectedTopicId`
   - `selectedActionId`
   - `topicWizardActive`
   - `graphViewActive` if `/world` and `/graph` fully replace it
4. Keep a smaller selection store only for true in-surface selection such as multi-select in the 3D world or graph selection highlights.
5. Remove stale comments that describe `/?skill=...`, `/?view=...`, and dialog-style editor closing.

Acceptance:

- `rg "useUrlState|updateUrl|replaceState|popstate|\\?skill|\\?agent|\\?team|graphViewActive|topicWizardActive" scenarios/prompt-manager/ui/src` returns no old navigation implementation, except intentional browser history tests or non-navigation uses with reviewed justification.

### Phase 7: Search-param state for tabs and highlights

1. Replace old pending tab state with route-local `useSearchParams`.
2. Team tabs should read/write `tab` and `subTab`.
3. Skill/agent/team highlight requests should read/write `hlFile`, `hlLine`, and `hlText`.
4. Decide whether clearing a handled highlight should use `{ replace: true }` to avoid adding a second history entry. Default to replace for one-shot consumed highlight state.
5. File/member deep links should use explicit query keys, for example `file` and `member`.

Acceptance:

- Tab changes update only search params for the current detail route.
- Browser Back from a tab change follows the chosen contract. If tab changes are considered page-local history, push. If they are ephemeral, replace. Document the decision in `route-paths.ts` tests.

### Phase 8: Validation and cleanup

1. Run static checks:

```bash
cd scenarios/prompt-manager/ui && pnpm run type-check
cd scenarios/prompt-manager/ui && pnpm run test
cd scenarios/prompt-manager/ui && pnpm run build
```

2. Run scenario-level validation:

```bash
cd scenarios/prompt-manager && make test
vrooli scenario test prompt-manager
```

3. Manual browser validation through the lifecycle system only:

```bash
cd scenarios/prompt-manager && make start
cd scenarios/prompt-manager && make logs
cd scenarios/prompt-manager && make stop
```

Manual flows:

- Open `/world`, click a skill, verify URL becomes `/skills/:skillId`.
- Browser Back returns to `/world`.
- Open `/graph`, click an entity, verify URL becomes the entity route and Back returns to `/graph`.
- Direct-load `/skills/:skillId`, click close, verify fallback is `/world`.
- Open `/teams/:teamId?tab=activity&subTab=decisions`, refresh, verify same team tab renders.
- Navigate cross-reference to a highlighted file/line, verify params are consumed according to the replace/push contract.
- On mobile width, opening an entity closes the sidebar, menu opens it, and close navigates back.

## Contract Decisions

- Canonical routes, not query keys, identify pages.
- Search params are allowed only for route-local UI state: tab, subTab, highlight, file/member focus.
- Entity IDs in route params are URL-encoded raw IDs. Do not invent slugs unless the API has stable slugs.
- Route helpers are the only module allowed to construct app paths.
- The shell owns shared data loading and sidebar state; pages own entity-specific rendering and route params.
- Form dirty state remains in editor stores. Navigation should store current changes before route transitions where existing behavior requires it, but it should not block route changes unless an explicit confirmation UX is implemented.
- Settings can remain a shell dialog/drawer unless a separate product decision makes it a route.

## Testing Plan

Unit tests:

- Route helper tests modeled after `swarm-manager/ui/src/app/routes/route-paths.test.ts`.
- `useAppBack` tests for in-app history and fallback replace behavior.
- `DetailPageHeader` tests modeled after swarm-manager's close/sidebar tests.
- Page route tests with `MemoryRouter` for every entity page.
- Sidebar callback tests that assert `useNavigate` receives route paths instead of store setter navigation.

Integration tests:

- Render `App` with memory history and verify `/world`, `/graph`, and each entity route select the correct page.
- Verify direct route params hydrate the correct editor data hook.
- Verify deleting or renaming an entity navigates to the correct fallback/new route.

Regression tests to delete or rewrite:

- `hooks/useUrlState.test.ts` must be removed.
- Any test asserting `window.history.replaceState` for entity selection must be replaced with route navigation assertions.

## Rollout/Validation Checklist

- [ ] `react-router-dom` dependency added with explicit approval before package installation.
- [ ] `App.tsx` uses `BrowserRouter` and route table.
- [ ] `AppShell` owns persistent sidebar and `<Outlet />`.
- [ ] All entity editors are reachable via direct routes.
- [ ] World and graph are direct routes.
- [ ] Sidebar and graph/world clicks call route helpers.
- [ ] Close and Escape use `useAppBack`.
- [ ] Old query URL adapter deleted.
- [ ] Selection store reduced to non-routing selection only.
- [ ] Route helper, header, app route, and sidebar navigation tests pass.
- [ ] Type check, test suite, build, and scenario test pass.
- [ ] Manual browser validation confirms browser Back/Forward history.

## Risks and Mitigations

- Risk: `SkillManagerLayout` is large and mixes unrelated responsibilities.
  Mitigation: Decompose by extracting shell-owned concerns first, then pages. Avoid editing behavior inside editor panels until route pages can pass existing props.

- Risk: Dirty-state handling could regress when route changes replace selected IDs.
  Mitigation: centralize route transition side effects in shell/page helpers and add tests that dirty state is stored before navigating to another entity.

- Risk: Direct-loaded detail routes may render before list data is loaded.
  Mitigation: keep existing editor data hooks and loading states; pages should fetch by route param and render current loading/error states.

- Risk: Tab/highlight behavior can create noisy history entries.
  Mitigation: explicitly decide push vs replace per search param family and test it.

- Risk: Mobile sidebar behavior is easy to conflate with close/back behavior.
  Mitigation: keep separate buttons and labels: menu opens sidebar, X closes page.

## Non-Goals / Prohibited Patterns

- No old query-param compatibility layer.
- No manual `window.history.pushState` or `replaceState` for app routing.
- No second router abstraction outside `react-router-dom`.
- No route decisions hidden inside Zustand setters.
- No broad visual redesign while doing the routing cutover.
- No new scenario lifecycle commands outside `make start`, `make test`, `make logs`, `make stop`, or `vrooli scenario test prompt-manager`.

## Definition of Done

- Entity detail pages in prompt-manager are first-class browser routes.
- Opening entity details pushes history.
- Browser Back/Forward works naturally between home surfaces and details.
- Direct entity URLs are refresh-safe.
- Close buttons and Escape use route back with a deterministic fallback.
- Old `useUrlState` query-navigation code and tests are gone.
- No dead compatibility code remains.
- All planned checks pass, or any failure is documented with exact command output and owner-visible next action.

