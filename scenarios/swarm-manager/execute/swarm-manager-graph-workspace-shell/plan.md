# Implementation Plan: Replace the Tabbed Shell with the Graph Workspace Shell

## 1. Purpose

Replace swarm-manager's 5-tab MainLayout (Backlog, Scenarios, Execution, Prompts, Settings) with a sidebar + graph canvas + inspector workspace at `/graph`. This is the application frame and interaction model — not about finalizing graph lens content.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read react-coherence ux
```

Additionally, the executing agent should read:
- Research conclusion: `swarm-manager backlog file-get --kind research --name swarm-manager-graph-workspace-contract --path conclusion.md`
- This plan's workshop rounds for settled decisions

## 3. Problem Statement

The current swarm-manager UI is a flat 5-tab CRUD shell (`MainLayout.tsx`) with page-based navigation. Entities (backlog items, scenarios, execution records, captures, initiatives) are siloed across tabs with no visual relationship between them. The research contract (`swarm-manager-graph-workspace-contract`) concluded that a graph-first workspace with sidebar + canvas + inspector is the target architecture.

This execute item builds the **shell** — the structural frame that hosts graph content. Lens-specific node rendering, edge logic, and inspector action panels are separate execute items (topology-impact-views, flow-ops-views).

## 4. Scope

### In Scope
- New `/graph` route with query params `?lens=topology|flow|operations&select=kind/name`
- Sidebar: activity-first unified feed with entity-type filter toggles
- Graph canvas: React Flow v12 + Dagre container, Zustand graph store with highlight/dim/hide query modes, viewport persistence via localStorage
- Inspector frame: desktop popover anchored to node (clamped to viewport bounds), mobile bottom sheet variant
- Header: title, lens switcher (segmented control), settings gear icon, agent-run status dropdown
- 301 redirects from legacy paths (client-side React Router redirects)
- Three layout modes: hierarchical (Dagre network-simplex), compact (Dagre tight-tree), grouped (custom lane-per-type)
- BFS neighborhood selection on node click (type-constrained traversal)
- Removal of the 5-tab navigation
- Settings and Prompts demoted to modal/drawer accessible via gear icon

### Out of Scope
- Lens-specific node/edge content (handled by topology-impact-views and flow-ops-views items)
- Graph data API endpoint (handled by graph-data-foundation item)
- SSE real-time streaming (handled by flow-ops-views item)
- Mobile-specific optimizations beyond basic responsiveness (handled by mobile-graph-interaction item)
- Legacy surface cleanup/removal (handled by remove-legacy-tabbed-surfaces chore)

## 5. Current Technical Context

### Key Files
- `scenarios/swarm-manager/ui/src/components/layout/MainLayout.tsx` — Current 5-tab shell (desktop header + mobile bottom nav)
- `scenarios/swarm-manager/ui/src/App.tsx` — React Router routes (7 routes under MainLayout)
- `scenarios/swarm-manager/ui/src/stores/` — 6 Zustand stores (backlog, scenarios, execution, agent-runs, backlog-form, capture)
- `scenarios/swarm-manager/ui/src/lib/feed.ts` — Smart backlog feed generation with attention reasons
- `scenarios/swarm-manager/ui/src/hooks/useKeyboardShortcuts.ts` — Tab navigation via keys 1-5
- `scenarios/swarm-manager/ui/src/services/` — 8 API service files

### Technology Stack
- React 18.3, Vite 5.4, TypeScript 5.6
- React Router 6.28
- Zustand 5.0 (localStorage-persisted stores)
- TailwindCSS 3.4
- Lucide React icons
- Vitest + React Testing Library (40 existing test files, co-located with source)

### New Dependencies Required
- `@xyflow/react` (React Flow v12) — graph canvas rendering
- `dagre` — graph layout algorithms
- `@types/dagre` — type definitions

## 6. Target End State

A single-page graph workspace at `/graph` that:
1. **Header** displays: workspace title, lens switcher (segmented control for topology/flow/operations), settings gear icon opening a drawer, agent-run status dropdown (migrated from current MainLayout)
2. **Sidebar** (left panel, 320px, fully collapsible to 0px) shows an activity-first unified feed with entity-type filter toggles, built on extended `lib/feed.ts` attention-reason logic. A floating toggle button restores the sidebar when collapsed. Collapse state persisted in localStorage.
3. **Graph canvas** (center) renders React Flow v12 with Dagre layouts, supports three layout modes (default: hierarchical for all lenses), and persists viewport to localStorage
4. **Inspector** (floating panel) opens on node selection — desktop: popover anchored to node position, clamped to viewport; mobile: bottom sheet
5. **Graph stores** (Zustand, split into graph-data + graph-ui) manage: nodes/edges/lens/filters (data store) and selectedNode/highlightState/layoutMode/viewport (UI store)
6. All legacy routes (`/backlog/:kind/:name`, `/scenarios/:name`, `/execution`, etc.) redirect to appropriate `/graph?...` URLs
7. Settings and Prompts accessible via gear icon → single drawer with internal tabs
8. Keyboard shortcuts: 1-3 switch lenses, L cycles layout modes, I toggles inspector, Esc deselects node, Ctrl+K preserved for host switcher

## 7. Implementation Strategy

### Phase 1: Foundation (graph stores + canvas container)
1. Install React Flow v12 (`@xyflow/react`) and `dagre` + `@types/dagre`
2. Create `src/surfaces/graph/` directory structure: `components/`, `hooks/`, `stores/`, `lib/`
3. Create graph-data store (`graphDataStore.ts`): nodes, edges, lens, filters
4. Create graph-ui store (`graphUIStore.ts`): selectedNode, highlightState, layoutMode, viewport
5. Create `GraphCanvas` component wrapping React Flow with Dagre layout
6. Implement three layout modes as Dagre config presets (default: hierarchical for all lenses)
7. Implement viewport persistence (localStorage via store middleware)
8. Stub node/edge rendering (placeholder nodes — real content comes from lens items)

### Phase 2: Shell frame (header + sidebar + inspector)
1. Create `GraphWorkspace` layout component (replaces MainLayout as the primary shell)
2. Build header with: title, lens switcher (segmented control), gear icon, agent-run dropdown (migrate from MainLayout)
3. Build sidebar with: unified feed (extend `lib/feed.ts` with multi-entity support), entity-type filter toggles, full hide/show behavior (0px collapsed ↔ 320px expanded) with floating toggle button, collapse state in localStorage
4. Build inspector frame: desktop floating popover with viewport clamping, mobile bottom sheet (reuse existing `components/ui/bottom-sheet` component)
5. Implement BFS neighborhood selection on node click (type-constrained traversal in graph-ui store)

### Phase 3: Routing + migration
1. Add `/graph` route with `lens` and `select` query params
2. Implement client-side 301 redirects from all legacy paths via React Router `<Navigate replace>`
3. Replace MainLayout with GraphWorkspace as the primary shell
4. Move Settings content to drawer behind gear icon (tabbed: Settings + Prompts)
5. Update keyboard shortcuts: replace tab-switching 1-5 with 1-3 for lenses, L for layout cycle, I for inspector toggle, Esc to deselect

### Phase 4: Data wiring + polish
1. Build client-side graph data assembler: map existing store data (backlog, scenarios, captures, execution, agent-runs) into React Flow nodes/edges format
2. Wire sidebar feed to live data (extended feed.ts)
3. Wire graph canvas to assembled data
4. Ensure inspector frame receives selected node and renders placeholder detail card
5. Test all legacy redirects
6. Verify responsive behavior (sidebar hide on mobile, inspector mode switching)

## 8. Contract Decisions

### Store Architecture (Workshop R1 d1 — settled: B)
Split into two Zustand stores:
- **graphDataStore**: nodes, edges, lens, filters. Owns data fetching and transformation.
- **graphUIStore**: selectedNode, highlightState (highlight/dim/hide query modes), layoutMode, viewport. Owns interaction state.
Rationale: Follows codebase pattern of small focused stores. Data changes don't trigger UI re-renders.

### Sidebar Feed (Workshop R1 d2 — settled: A)
Extend existing `lib/feed.ts` with multi-entity support (captures, scenarios, execution records alongside backlog items). Reuses proven attention-reason logic.

### Settings/Prompts Demotion (Workshop R1 d3 — settled: A)
Single gear icon → one drawer with internal tabs for Settings and Prompts. Keeps both accessible from the same entry point.

### Component Organization (Workshop R1 d4 — settled: A)
New `src/surfaces/graph/` directory with subdirectories: `components/`, `hooks/`, `stores/`, `lib/`. Shared primitives stay in existing `components/ui/`. Follows react-coherence surfaces pattern.

### Data Bootstrapping (Workshop R1 d5 — settled: A)
Assemble graph data client-side from existing stores/APIs. Throwaway mapper code that gets replaced when the dedicated graph API lands. Makes the shell immediately functional and testable with real data.

### Test Strategy (Workshop R2 d1 — settled: A)
Store-first testing approach. Unit test graphDataStore and graphUIStore extensively (selection, highlight, layout, viewport persistence). Component tests verify correct props reach React Flow and that UI controls dispatch correct store actions — but do not assert on React Flow's internal rendering. BFS traversal and layout utilities get dedicated unit tests. This avoids fighting React Flow's rendering internals while providing high confidence in behavior.

### Keyboard Shortcut Scheme (Workshop R2 d2 — settled: A)
Number keys for lenses (1=Topology, 2=Flow, 3=Operations), letter keys for actions (L=cycle layout mode, I=toggle inspector), Esc=deselect/close. Ctrl+K preserved for host switcher. Rewrite `useKeyboardShortcuts` hook for graph context.

### Sidebar Collapse Behavior (Workshop R2 d3 — settled: B)
Full hide (0px) on both desktop and mobile. Floating toggle button to restore the sidebar. No icon rail — maximizes graph canvas space. Collapse state persisted in localStorage. Mobile: sidebar hidden by default, toggle button opens as overlay. Desktop: same behavior.

### Default Layout Mode (Workshop R2 d4 — settled: A)
Same default layout (hierarchical / Dagre network-simplex) for all three lenses. User can switch manually; layout preference persists per-lens in localStorage so they only set it once.

### Routing Contract
- URL: `/graph?lens=topology|flow|operations&select=kind/name`
- Default lens: `topology`
- Legacy redirects: client-side via React Router `<Navigate>` with replace

### Inspector Contract
- Desktop: floating popover anchored to selected node's position, clamped to viewport bounds (min 16px margin from edges)
- Mobile: bottom sheet (reuse existing `components/ui/bottom-sheet` component)
- Trigger: node click or `?select=kind/name` URL param
- Dismiss: click canvas background, press Esc, or click close button

### Sidebar Contract
- Width: 320px expanded, 0px collapsed (full hide on both desktop and mobile)
- Restore: floating toggle button visible when sidebar is collapsed
- Feed source: extended `lib/feed.ts` producing unified feed items
- Filter toggles: per entity type (capture, backlog, scenario, execution)
- Collapse state: persisted in localStorage

## 9. Testing Plan

### Test Approach: Store-First (Workshop R2 d1)
Focus testing effort on store logic and utility functions. Component tests verify wiring (correct props, correct store dispatch) without asserting on React Flow internals.

### Unit Tests (co-located with source, matching existing pattern)
- **graphDataStore.test.ts**: node/edge CRUD, lens switching, filter application, data assembly from mocked store snapshots
- **graphUIStore.test.ts**: selection state, highlight/dim/hide transitions, layout mode cycling, viewport persistence round-trip
- **bfs-selection.test.ts**: BFS traversal (type-constrained), depth limits, disconnected graph handling, cycle resilience
- **layout-utils.test.ts**: Dagre config generation for each layout mode, edge case (empty graph, single node, disconnected components)
- **feed.test.ts extensions**: new entity types in feed (captures with classification status, scenarios with runtime status, execution records)

### Component Tests (React Testing Library)
- **GraphWorkspace.test.tsx**: renders header + sidebar + canvas regions, sidebar toggle works (0px ↔ 320px), responsive breakpoints
- **LensSwitcher.test.tsx**: switching updates URL param, active state visual
- **Sidebar.test.tsx**: filter toggles show/hide entity types, feed renders items in priority order, floating toggle button visible when collapsed
- **Inspector.test.tsx**: opens on node selection, closes on Esc/background click, renders detail card placeholder
- **SettingsDrawer.test.tsx**: opens from gear icon, contains Settings + Prompts tabs, tab switching works

### Integration Tests
- **Legacy redirect suite**: each of the 7 old routes maps to the correct new `/graph?...` URL (parametrized test)
- **URL ↔ state sync**: `?lens=` and `?select=` params correctly drive store state on initial load and on navigation
- **Keyboard shortcuts**: 1-3 switch lenses, L cycles layout, I toggles inspector, Esc deselects

### What NOT to test in this item
- Lens-specific node rendering (separate items)
- Graph data API (separate item)
- SSE real-time updates (separate item)
- Per-node-type inspector action panels (separate items)

## 10. Rollout / Validation Checklist

- [ ] `/graph` loads with sidebar, canvas, and header
- [ ] Lens switcher changes `?lens=` param and re-renders
- [ ] `?select=kind/name` auto-selects a node and opens inspector
- [ ] All 7 legacy routes redirect correctly
- [ ] Settings accessible via gear icon → drawer with Settings tab
- [ ] Prompts accessible via gear icon → drawer with Prompts tab
- [ ] Agent-run dropdown functional in new header
- [ ] Sidebar shows unified feed with filter toggles
- [ ] Sidebar collapses to 0px with floating toggle to restore
- [ ] Three layout modes produce visually distinct arrangements
- [ ] Default layout is hierarchical for all lenses
- [ ] Viewport persists across page refreshes
- [ ] Mobile: sidebar hidden by default, inspector shows as bottom sheet
- [ ] Keyboard shortcuts: 1-3 lenses, L layout, I inspector, Esc deselect
- [ ] No 5-tab navigation remnants visible

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| React Flow v12 bundle size | Increased initial load | Tree-shake, lazy-load graph route via React.lazy + Suspense |
| Graph store complexity | Two stores still accumulate state | Clear ownership boundary: data store = what to render, UI store = how to render |
| Sidebar feed performance | Too many re-renders on data updates | `useShallow()` selectors, memo on feed items, debounce filter toggles |
| Inspector viewport clamping | Edge cases at screen boundaries | Clamp algorithm with 16px min-margin, reposition on window resize, unit test edge cases |
| Legacy redirect coverage | Missed deep links from agent-manager | Exhaustive parametrized redirect test (all 7 routes × typical params) |
| Settings/Prompts discoverability | Reduced discoverability behind gear icon | Gear icon badge/indicator when settings need attention |
| Client-side data assembly perf | Mapping 6 stores into graph format on every change | Memoize assembly with useMemo/selector, only recompute on store version change |
| Dagre layout on large graphs | Layout computation blocks main thread | Web Worker for Dagre if >200 nodes, otherwise synchronous is fine |
| Existing test breakage | MainLayout.test.tsx and page tests assume tabbed navigation | Update tests as part of Phase 3; redirect tests replace old route tests |
| Sidebar filter inaccessibility when collapsed | Full-hide means no filter access without reopening | Floating toggle is fast to click; filter state persists so users set-and-forget |

## 12. Non-goals / Prohibited Patterns

- No compatibility layer — this is a hard-cut replacement
- No prompt-manager dependency — build graph independently, adopt patterns only
- No server-side redirects — all legacy path handling is client-side React Router
- No lens-specific node content — this item builds the frame, not the per-lens rendering
- No new API endpoints — this item consumes existing APIs; the graph API is a separate item
- No legacy tabbed navigation preserved "just in case"
- No `lib/` folder in the surface — use `surfaces/graph/lib/` only for graph-specific utilities

## 13. Definition of Done

1. `/graph` is the default route and renders the full workspace shell
2. Sidebar displays activity feed with working filter toggles
3. Sidebar collapses to 0px (full hide) with floating toggle to restore, collapse state persisted
4. Graph canvas renders with React Flow + Dagre (with client-side assembled data from existing stores)
5. Inspector opens on node selection (desktop popover, mobile bottom sheet)
6. All 7 legacy routes produce correct redirects
7. Settings and Prompts accessible from gear icon in tabbed drawer
8. Agent-run dropdown works in new header
9. Three layout modes selectable and functional, default hierarchical for all lenses
10. Keyboard shortcuts: 1-3 lenses, L layout, I inspector, Esc deselect, Ctrl+K host switcher
11. All new components have co-located test files following store-first testing approach
12. Legacy redirect integration test passes for all 7 routes
13. Graph store unit tests cover selection, highlight, layout, viewport persistence
14. No regressions in existing API interactions
15. No remnants of 5-tab navigation in the active UI
