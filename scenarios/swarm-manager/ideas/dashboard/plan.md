# Dashboard — Implementation Plan

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## 1. Purpose

Add a Dashboard landing page to the swarm-manager UI that provides at-a-glance situational awareness: backlog pipeline health, initiative progress, scenario status, active execution runs, and dependency graph visualization. Currently the UI opens directly to the Backlog tab — the Dashboard becomes the new default landing page, surfacing the most actionable information across all tabs in one view.

## 2. Problem Statement

Operators managing the Vrooli scenario ecosystem must currently navigate between 4 separate tabs (Backlog, Scenarios, Execution, Prompts) to build a mental model of system state. There is no single view that answers "what's happening right now?" or "what needs my attention?" The backend already exposes a `/api/v1/overview` endpoint with aggregated data (backlog summary stats, initiatives with rollup, dependency graph edges/blocked/unblocked), but no UI consumes it.

## 3. Scope

### In Scope
- New `DashboardPage` component as the default landing route (`/`)
- Summary stat cards (total items, items by status, items by kind, active initiatives)
- Initiative progress cards with rollup status
- Blocked/unblocked item highlights from dependency graph
- Recent execution activity feed (pending, running, completed, failed)
- Quick-action links to navigate to specific backlog items, scenarios, or execution runs
- Responsive layout (desktop and mobile)

### Out of Scope
- New backend APIs (the overview endpoint already exists; execution data is available from existing endpoints)
- Real-time WebSocket updates (use polling or on-mount fetch for v1)
- Customizable widget layout or user-configurable dashboard
- Historical trend charts or analytics (can be a follow-up)
- Modifications to existing pages

## 4. Current Technical Context

### Key Files/Components
- **Overview API**: `api/internal/overview/service.go` — returns `OverviewResponse` with items, initiatives, dependency_graph, summary
- **Overview endpoint**: `GET /api/v1/overview` registered in `api/internal/overview/handler.go`
- **Execution API**: Existing execution endpoints in `api/internal/execution/`
- **UI routing**: `ui/src/App.tsx` — currently `<Route index element={<Navigate to="/backlog" replace />} />`
- **UI layout**: `ui/src/components/layout/MainLayout.tsx` — tabbed navigation (Backlog, Scenarios, Execution, Prompts, Settings)
- **Existing patterns**: `BacklogPage.tsx`, `ScenariosPage.tsx`, `ExecutionPage.tsx` for page structure reference

### API Response Shape (overview)
```json
{
  "items": [...],
  "initiatives": [{ "initiative": {...}, "rollup": {...} }],
  "dependency_graph": { "edges": [[from, to]], "unblocked": [...], "blocked": [...] },
  "summary": { "total_items": N, "items_by_status": {...}, "items_by_kind": {...}, "active_initiatives": N }
}
```

## 5. Target End State

A Dashboard page at `/` that:
1. Shows summary stat cards (total backlog items, by-status breakdown, active initiatives)
2. Lists blocked items needing attention and unblocked items ready to process
3. Shows initiative progress with rollup percentages
4. Displays recent execution activity (last N runs with status)
5. Provides quick-action links to drill into any item
6. Renders on both desktop (card grid) and mobile (stacked cards)

## 6. Implementation Strategy

### Phase 1: Core Dashboard Page
- Create `DashboardPage.tsx` with data fetching from `/api/v1/overview`
- Create widget components: `SummaryCards`, `BlockedItems`, `InitiativeProgress`
- Update `App.tsx` routing: change index route from redirect-to-backlog to Dashboard
- Add "Dashboard" tab to navigation layout

### Phase 2: Execution Activity Feed
- Add `RecentActivity` widget fetching from execution endpoints
- Show last N execution runs with status badges and timestamps
- Link each run to the Execution page details

### Phase 3: Polish
- Loading skeletons for each widget
- Error states per widget (partial failure tolerance)
- Mobile-responsive layout adjustments
- Auto-refresh on configurable interval (default: 60s)

## 7. Contract Decisions

- **Route**: `GET /` renders `DashboardPage` (replaces redirect to `/backlog`)
- **Navigation**: Add "Dashboard" as first tab, shifting existing tabs right
- **Data source**: `GET /api/v1/overview` for primary data; execution endpoints for activity feed
- **No new API endpoints** in v1 — compose from existing endpoints client-side

## 8. Testing Plan

- Unit tests for each widget component (render with mock data, loading state, error state)
- Integration test for DashboardPage data fetching
- Navigation test: verify `/` loads Dashboard, tab navigation works
- Mobile viewport test: verify responsive layout

## 9. Rollout / Validation Checklist

- [ ] `DashboardPage` renders with real data from overview API
- [ ] All stat cards show correct counts
- [ ] Blocked/unblocked items link to correct backlog detail pages
- [ ] Initiative progress reflects actual rollup data
- [ ] "Dashboard" tab is active on `/` route
- [ ] Existing tab routes still work
- [ ] Mobile layout is usable

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Overview API returns too much data for dashboard use | Low | Medium | Use summary fields; lazy-load full item lists only on expand |
| Adding a 6th tab crowds navigation on mobile | Medium | Medium | Workshop decision d3 addresses this — may use icon-only on mobile |
| Execution endpoints may not return data in ideal format for activity feed | Low | Low | Adapt client-side; defer API changes to follow-up |

## 11. Non-goals / Prohibited Patterns

- Do not add WebSocket infrastructure for v1
- Do not create new backend endpoints — compose from existing APIs
- Do not add user preferences/customization — keep it simple
- Do not duplicate data display that's already well-served by existing pages

## 12. Definition of Done

- Dashboard page loads at `/` showing overview data
- Navigation includes Dashboard as the first tab
- All widgets render correctly with real data
- Tests pass for new components
- No regressions on existing pages
- Responsive on mobile viewports
