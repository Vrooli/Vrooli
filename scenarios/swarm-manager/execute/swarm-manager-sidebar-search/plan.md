# Plan: Add Search, Filtering, Tabs, and Full-Viewport Mobile to the Graph Sidebar

## Purpose

Transform the swarm-manager graph sidebar from a read-only activity feed into a full navigation surface with search, filtering, sorting, tabbed views, and a native-feeling mobile experience.

## Required Reading

```bash
prompt-manager skill read react-coherence ux
```

## Problem Statement

The current sidebar (`surfaces/graph/components/Sidebar.tsx`) is a 152-line component that renders a flat `FeedItem[]` list built by `lib/feed.ts`. It merges captures and backlog items with priority sorting but provides no way to:

- **Search** for specific items by text
- **Filter** by entity type, status, priority, or tags
- **Sort** by different dimensions
- **Browse** specific entity types (initiatives, executions, captures independently)

On mobile, the sidebar uses `w-[85vw] max-w-[320px]` — a narrow overlay that doesn't feel like a native app page.

## Scope

### In Scope
- Search bar with debounced client-side filtering
- Tabbed navigation (Activity, Backlog, Captures, Initiatives, Executions)
- Per-tab filter and sort controls
- Full-viewport mobile layout
- URL state persistence for active tab and active tab's filters
- Graph node selection integration from all tabs
- Simple text-with-icon empty states per tab
- Loading skeleton for initiative-store initial fetch

### Out of Scope
- Semantic/vector search (Qdrant integration)
- API-side search/filtering changes (client-side filtering only for now)
- Changes to the graph canvas itself
- New API endpoints
- Settings drawer modifications
- Illustrated empty states or contextual action CTAs

## Current Technical Context

### Key Files
| File | Role |
|------|------|
| `surfaces/graph/components/Sidebar.tsx` | Current sidebar — 152 lines, flat feed rendering |
| `lib/feed.ts` | Builds `FeedItem[]` from captures + backlog items |
| `stores/backlog-store.ts` | Zustand store, persists up to 200 backlog items |
| `stores/capture-store.ts` | Zustand store, persists up to 100 captures |
| `stores/execution-store.ts` | Zustand store for execution records with polling |
| `services/initiative-service.ts` | API service for initiatives (list, get, files) |
| `surfaces/graph/stores/graph-ui-store.ts` | Sidebar collapsed state, selected node |
| `surfaces/graph/stores/graph-data-store.ts` | Graph projection, entity filters, per-lens settings |
| `components/ui/search-bar.tsx` | Existing reusable search input with Search icon |
| `components/ui/input.tsx` | CVA-based input with icon slots |
| `surfaces/graph/lib/node-id-parser.ts` | `buildBacklogNodeId`, `buildExecutionNodeId`, etc. |
| `surfaces/graph/components/GraphWorkspace.tsx` | `handleSidebarItemClick` → `selectNode` + URL param |
| `config/index.ts` | `searchDebounceMs: 300` default |

### Data Sources Per Tab
| Tab | Source | Notes |
|-----|--------|-------|
| Activity | `lib/feed.ts` → `buildFeed()` | Existing — merges captures + backlog with priority sort |
| Backlog | `backlog-store` | Already loaded — `items[]` with status, kind, priority, tags |
| Captures | `capture-store` | Already loaded — `captures[]` with status |
| Initiatives | `initiative-store` (new) | Zustand store wrapping `initiative-service.list()` — returns `InitiativeWithRollup[]` |
| Executions | `execution-store` | Already loaded — `items[]` with status, mode, timing |

### Existing Patterns
- **Tab pattern**: SettingsDrawer uses inline `useState<Tab>` with `border-b-2` active indicator (cyan-400)
- **Search**: `SearchBar` component exists in `components/ui/search-bar.tsx`
- **Node selection**: `onItemClick(nodeId)` → `selectNode()` + `setSearchParams("select", nodeId)`
- **Store pattern**: Zustand with persistence, selector-based reads
- **URL state**: `useSearchParams` from React Router already used in GraphWorkspace
- **Node ID builders**: `buildBacklogNodeId(kind, name)`, `buildExecutionNodeId(id)`, `buildActivityNodeId(id)`

### Types
- `BacklogKind`: `"idea" | "research" | "fix" | "execute" | "chore"`
- `BacklogStatus`: `"backlog" | "researching" | "ready" | "queued" | "in_progress" | "completed" | "failed" | "archived"`
- `CaptureStatus`: `"classifying" | "classified" | "failed"`
- `ExecutionStatus`: `"pending" | "scheduled" | "starting" | "running" | "needs_review" | "validating" | "needs_fixup" | "completed" | "failed" | "canceled"`
- `ExecutionMode`: `"manual" | "scheduled" | "yolo"`
- `InitiativeStatus`: `"active" | "completed" | "archived"`
- `FeedItem`: discriminated union with `type: "capture" | "attention" | "backlog"`

## Target End State

The sidebar becomes a multi-tab navigation surface:

```
┌─────────────────────┐
│ 🔍 Search...        │  ← Persistent search bar
├─────────────────────┤
│ Activity│Backlog│Cap.│  ← Horizontally scrollable tabs
├─────────────────────┤
│ ▼ Filters  ▲ Sort   │  ← Collapsible filter + sort controls
├─────────────────────┤
│                     │
│   [Feed/List items] │  ← Tab-specific content
│                     │
└─────────────────────┘
```

- **Desktop**: 320px width preserved, controls stack vertically
- **Mobile**: Full viewport width (`w-full`), app-like full-screen experience
- **All tabs**: Clicking an item selects the corresponding graph node

## Contract Decisions

### State Management
- **Decision**: `useReducer` in `Sidebar.tsx` for all sidebar UI state (active tab, search query, per-tab filters, per-tab sort). No new Zustand store for sidebar UI.
- **Rationale**: All state is local to the sidebar subtree. URL sync via `useSearchParams` at this level. Follows react-coherence "local state first" principle.

### Component Structure
- **Decision**: Separate file per tab component under `surfaces/graph/components/sidebar/`.
- **Layout**: `Sidebar.tsx` orchestrates search → tabs → filters → content. Each tab (e.g., `BacklogTab.tsx`) handles its own list rendering and metadata display.

### Data Sources
- **Activity tab**: Use existing `buildFeed()` from `lib/feed.ts`
- **Backlog tab**: Read directly from `backlog-store`
- **Captures tab**: Read directly from `capture-store`
- **Executions tab**: Read directly from `execution-store`
- **Initiatives tab**: Create a dedicated `initiative-store` (Zustand) wrapping `initiative-service.list()`, following the same pattern as `execution-store`. Provides consistent data lifecycle, caching, selector-based reads, and refresh logic across all entity tabs. The `initiative-service.ts` API layer already exists — the store adds reactive access and caching.

### URL Params
- **Encoding**: Individual params with compact keys. Only non-default values serialized.
- **Params**: `?stab=backlog` (sidebar tab), `?sst=backlog,ready` (status filter), `?spr=1-3` (priority range), `?ssort=priority` (sort field), `?sdir=asc` (sort direction)
- **Prefix**: All sidebar params use `s` prefix to avoid collision with existing `select`, `lens`, `focus` params.

### Filter Persistence
- **Decision**: Filters persist per-tab in the reducer state, but the URL only reflects the **active tab's** filters.
- **Behavior**: Switching tabs preserves each tab's filter/sort state in the reducer for the duration of the session. The URL updates to show only the visible tab's filters. On page reload, only the active tab's filters survive (restored from URL); other tabs reset to defaults.
- **Rationale**: Keeps the URL simple and human-readable while providing good in-session UX. Users switching between tabs don't lose their filter context, but the URL doesn't become unwieldy with all tabs' state encoded simultaneously.

### Empty & Loading States
- **Decision**: Simple text-only empty states with a centered icon + short message per tab (e.g., "No backlog items match your filters"). Loading state for initiative-store uses a pulsing skeleton matching item card height.
- **Rationale**: Fast to implement, visually consistent across tabs. Illustrated empty states with CTAs are out of scope for this item.

## Implementation Strategy

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

### Phase 1: Sidebar Skeleton & Tabs
1. Create `surfaces/graph/components/sidebar/` directory
2. Extract current feed rendering into `ActivityTab.tsx`
3. Create `SidebarTabs.tsx` with horizontal scroll tab bar and tab switching
4. Create `useSidebarState` reducer hook with tab, search, per-tab filter, per-tab sort state
5. Refactor `Sidebar.tsx` to compose: search → tabs → filter bar → content
6. Add URL param sync for active tab (`?stab=backlog`) and active tab's filters

### Phase 2: Search
1. Integrate existing `SearchBar` component at top of sidebar
2. Create `useSidebarSearch` hook with debounced client-side filtering (uses `searchDebounceMs`)
3. Wire search to filter the active tab's content
4. Search targets: title/name/description for backlog; text for captures; name for initiatives; backlogName/status for executions

### Phase 3: Entity Tabs
1. **BacklogTab.tsx**: List backlog items with kind icons, status badges, priority indicators. Click → `buildBacklogNodeId(kind, name)` → `onItemClick`
2. **CapturesTab.tsx**: List captures with classification status. Click → `capture/{id}` → `onItemClick`
3. **InitiativesTab.tsx**: Create `initiative-store.ts` (Zustand) wrapping `initiative-service.list()`. List initiatives with rollup counts (total/completed/in_progress/failed/pending). Click → `initiative/{name}` → `onItemClick`
4. **ExecutionsTab.tsx**: List executions from `execution-store` with status (running/completed/failed), mode, timing info. Click → `buildExecutionNodeId(id)` → `onItemClick`

### Phase 4: Filtering & Sorting
1. Create `FilterBar.tsx` — collapsible section below tabs
2. Per-tab filter configs:
   - Backlog: status, priority range, kind, tags (chip-style toggles)
   - Captures: classification status
   - Executions: status, mode
   - Initiatives: status (active/completed/archived)
3. Sort controls: priority, recency, status, alphabetical with direction toggle
4. URL param sync: serialize active tab's filter/sort state only; restore on page load

### Phase 5: Mobile Full-Viewport
1. Change mobile width from `w-[85vw] max-w-[320px]` to `w-full`
2. Ensure search input `font-size: 16px` to prevent iOS zoom
3. Add `touch-action: manipulation` to sidebar container
4. Verify graph is non-interactive when sidebar is open on mobile

### Phase 6: Polish & Integration
1. Verify all tabs integrate with `onItemClick` → graph node selection
2. Test URL state persistence (tab, active tab's filters, sort)
3. Ensure back-button works correctly with URL state
4. Keyboard accessibility for tabs and filters
5. Empty states for each tab: centered icon + short descriptive message
6. Loading skeleton for initiative-store initial fetch (pulsing skeleton matching item card height)

### Final: Cleanup & Verification
- Run type checking (`npx tsc --noEmit`) and fix ALL errors, even pre-existing
- Run linter (`eslint`) and fix ALL warnings in modified files
- Run unit tests and fix any failures
- `vrooli scenario restart swarm-manager`
- Verify health: `curl -s http://localhost:<port>/health`

## Testing Plan

### Unit Tests
1. **`useSidebarState` reducer**: Test tab switching, per-tab filter toggling, sort changes, search query updates, URL sync serialization/deserialization (only active tab's filters in URL)
2. **Search filtering logic**: Test text matching across different entity fields (title, description, tags for backlog; text for captures; name for initiatives)
3. **Filter logic per tab**: Test each filter combination (status, priority range, kind, tags for backlog; classification status for captures; status/mode for executions; status for initiatives)
4. **Sort logic**: Test each sort dimension (priority, recency, status, alphabetical) with ascending/descending
5. **Initiative store**: Test fetch, caching, error handling following execution-store patterns

### Component Tests (Vitest + Testing Library)
1. **SidebarTabs**: Renders all 5 tabs, switches active tab on click, highlights active tab
2. **FilterBar**: Renders correct filters per active tab, collapses/expands, chip toggles work
3. **Each entity tab**: Renders items with correct metadata, calls onItemClick with correct node ID
4. **SearchBar integration**: Debounces input, filters displayed items
5. **Empty states**: Each tab renders correct icon + message when no items / no search results
6. **Loading state**: InitiativesTab shows pulsing skeleton before data loads

### Integration Tests
1. **URL state roundtrip**: Set URL params → sidebar reflects correct tab/filters. Change sidebar → URL updates. Only active tab's filters appear in URL.
2. **Tab switch filter persistence**: Set filters on Backlog, switch to Captures, switch back → Backlog filters preserved in reducer. Reload page → only active tab's filters survive.
3. **Graph node selection**: Click sidebar item → correct node selected in graph-ui-store
4. **Mobile layout**: Sidebar renders full-width on mobile viewport

## Rollout / Validation Checklist

- [ ] All 5 tabs render correct content
- [ ] Search filters items within the active tab
- [ ] Filter controls work per-tab with correct options
- [ ] Sort controls change item ordering
- [ ] Filters persist per-tab when switching tabs (session only)
- [ ] URL only encodes active tab's filters; other tabs reset on reload
- [ ] Mobile: sidebar is full-width, no input zoom, app-like feel
- [ ] Desktop: sidebar remains 320px
- [ ] Clicking items selects graph nodes in all tabs
- [ ] URL params persist tab, filters, and sort state for active tab
- [ ] Back button navigates tab/filter history correctly
- [ ] No regressions to existing feed behavior (Activity tab)
- [ ] Empty states: centered icon + message when no items match search/filters
- [ ] Initiative store fetches and caches data correctly
- [ ] Loading skeleton shown during initiative data fetch

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Large item lists cause scroll jank | Medium | Medium | Virtualize lists if >100 items; start without virtualization |
| Filter/sort state in URL becomes unwieldy | Low | Low | Use compact param encoding with `s` prefix; only persist active tab's non-default values |
| Initiative API latency on first load | Medium | Medium | Loading skeleton in InitiativesTab; initiative-store caches after first fetch |
| Tab bar takes too much vertical space on short screens | Medium | Medium | Compact tab styling; collapsible filters help |
| URL param collisions with existing params | Low | High | All sidebar params use `s` prefix; existing params (`select`, `lens`, `focus`) are unprefixed |
| Per-tab filter state lost on page reload for inactive tabs | Low | Low | Acceptable tradeoff — URL simplicity outweighs edge case of reloading with multi-tab filters active |

## Non-goals / Prohibited Patterns

- No Qdrant/semantic search integration
- No API changes (client-side filtering only, except initiative-store which uses existing API)
- No changes to the graph canvas itself
- No custom tab component library — use inline tab pattern matching SettingsDrawer
- No new Zustand stores except `initiative-store` (all other data sources exist)
- No settings drawer modifications
- No illustrated empty states or action CTAs in empty states
- No compatibility shims or legacy wrappers

## Definition of Done

1. All 5 sidebar tabs functional with correct entity-specific content
2. Search bar filters active tab content with debounce
3. Per-tab filter controls with chip-style toggles
4. Sort controls with direction toggle
5. Mobile sidebar is full viewport width with no input zoom
6. Graph node selection works from all tabs
7. URL state preserves tab + active tab's filter + sort across navigation
8. Filters persist per-tab in reducer during session; only active tab's filters in URL
9. Simple text-with-icon empty states for all tabs
10. Loading skeleton for initiative-store initial fetch
11. No regressions to existing Activity feed behavior
12. Unit tests for reducer, search, filter, and sort logic
13. Component tests for tabs, filter bar, entity tabs, empty states, and loading states
14. All lint, type, and test issues fixed (including pre-existing)
15. Scenario restarted and healthy
