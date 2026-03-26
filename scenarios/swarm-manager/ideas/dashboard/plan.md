# Dashboard — Implementation Plan

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement react-coherence ux
```

## 1. Purpose

Add a Dashboard landing page to the swarm-manager UI that provides at-a-glance situational awareness: backlog pipeline health, initiative progress, blocked items, and recent execution activity. Currently the UI opens directly to the Backlog tab — the Dashboard becomes the new default landing page, surfacing the most actionable information across all tabs in one view.

## 2. Problem Statement

Operators managing the Vrooli scenario ecosystem must currently navigate between 5 separate tabs (Backlog, Scenarios, Execution, Prompts, Settings) to build a mental model of system state. There is no single view that answers "what's happening right now?" or "what needs my attention?" The backend already exposes a `GET /api/v1/overview` endpoint with aggregated data (backlog summary stats, initiatives with rollup, dependency graph edges/blocked/unblocked), but no UI consumes it.

## 3. Scope

### In Scope
- New `DashboardPage` component as the default landing route (`/`)
- Summary stat cards (total items, items by status, items by kind, active initiatives)
- Initiative progress cards with rollup status
- Blocked/unblocked item highlights from dependency graph
- Recent execution activity feed (last N runs with status badges)
- Quick-action links to drill into specific backlog items, scenarios, or execution runs
- Responsive layout (2-column grid desktop, single-column mobile)
- Auto-poll every 60 seconds with manual refresh button
- Navigation: Dashboard as new first tab (6 tabs total) — **pending d1 round 2 on mobile nav approach**

### Out of Scope
- New backend APIs (the overview endpoint already exists; execution data is available from existing endpoints)
- Real-time WebSocket or SSE updates (use polling for v1)
- Customizable widget layout or user-configurable dashboard
- Historical trend charts or analytics (follow-up)
- Modifications to existing pages (other than routing/nav changes)
- Prompts/Settings page merge — **revisited in round 2 d1**
- Dependency graph visualization (follow-up)

## 4. Current Technical Context

### Key Files/Components
- **Overview API**: `api/internal/overview/service.go` — returns `OverviewResponse` with items, initiatives, dependency_graph, summary
- **Overview endpoint**: `GET /api/v1/overview` registered in `api/internal/overview/handler.go`
- **Execution API**: Existing execution endpoints in `api/internal/execution/`
- **UI routing**: `ui/src/App.tsx` — currently `<Route index element={<Navigate to="/backlog" replace />} />`
- **UI layout**: `ui/src/components/layout/MainLayout.tsx` — tabbed navigation with keyboard shortcuts (1-5)
- **API endpoints**: `ui/src/lib/api-endpoints.ts` — **NOTE: overview endpoint NOT yet registered here**
- **Existing patterns**: `BacklogPage.tsx`, `ScenariosPage.tsx`, `ExecutionPage.tsx` for page structure reference
- **Shared components**: `Card`, `StatusLegend`, `ErrorState`, `PageLoadingState`, `InlineLoadingIndicator`, `ResponsiveList` in `ui/src/components/ui/`
- **Status colors/icons**: `BACKLOG_STATUS_COLORS`, `SCENARIO_STATUS_ICONS` in `ui/src/types/constants.ts`

### API Response Shape (overview)
```json
{
  "items": [...],
  "initiatives": [{ "initiative": {...}, "rollup": {...} }],
  "dependency_graph": { "edges": [[from, to]], "unblocked": [...], "blocked": [...] },
  "summary": { "total_items": N, "items_by_status": {...}, "items_by_kind": {...}, "active_initiatives": N }
}
```

### Data Fetching Patterns in Use
- **Zustand stores**: Used by BacklogPage, ScenariosPage, ExecutionPage for pages with complex local state and mutations
- **React Query (useQuery)**: Used by PromptsPage, SettingsPage for simpler read-heavy pages
- **Polling**: ExecutionPage auto-refreshes every 6 seconds; MainLayout refreshes active runs every 5 seconds

## 5. Target End State

A Dashboard page at `/` that:
1. Shows summary stat cards (total backlog items, by-status breakdown, by-kind breakdown, active initiatives)
2. Lists blocked items needing attention and unblocked items ready to process
3. Shows initiative progress with rollup percentages
4. Displays recent execution activity (last N runs with status badges and timestamps)
5. Provides quick-action links to drill into any item
6. Renders on both desktop (2-column card grid) and mobile (stacked cards)
7. Auto-polls every 60 seconds with a manual refresh button

## 6. Implementation Strategy

### Phase 1: Data Layer & Core Page
- Add `overview` endpoint to `ui/src/lib/api-endpoints.ts`
- Create `ui/src/services/overview-service.ts` with typed fetch for `OverviewResponse`
- Create `DashboardPage.tsx` using `useQuery` for overview data + separate `useQuery` for recent executions
- Configure `refetchInterval: 60000` on both queries for auto-polling
- Add manual refresh button that calls `queryClient.invalidateQueries`

### Phase 2: Widget Components
- `SummaryCards` — stat cards for total items, by-status, by-kind, active initiatives
- `BlockedItems` — list of blocked items from dependency graph with links to backlog detail
- `InitiativeProgress` — initiative cards showing rollup status/percentage
- `RecentActivity` — last N execution runs with status badges, timestamps, and links

### Phase 3: Navigation & Routing
- Add "Dashboard" as first entry in `MainLayout.tsx` tabs array (icon: `LayoutDashboard` from lucide-react)
- Update `App.tsx`: change index route to `<DashboardPage />`, add `/dashboard` path
- Update keyboard shortcuts: Dashboard=1, shift existing tabs to 2-6
- Mobile navigation: **pending d1 round 2** — likely icon-only bottom nav for 6 tabs

### Phase 4: Polish & Error Handling
- Loading skeletons for each widget (use `PageLoadingState` or custom skeleton)
- Per-widget error states using `ErrorState` component (partial failure tolerance — one widget failing doesn't break others)
- Mobile-responsive layout: CSS Grid with `grid-cols-1 md:grid-cols-2` for 2-column desktop / 1-column mobile
- Refresh indicator showing last-fetched timestamp

### Final: Cleanup & Verification
- Run type checking (`npx tsc --noEmit`) and fix ALL errors, even pre-existing
- Run linter (`eslint`) and fix ALL warnings in modified files
- Run unit tests and fix any failures
- `vrooli scenario restart swarm-manager`
- Verify health: `curl -s http://localhost:<port>/health`

## 7. Contract Decisions

| Decision | Choice | Source |
|----------|--------|--------|
| Route | `GET /` renders `DashboardPage` (replaces redirect to `/backlog`) | Round 1 |
| Navigation | Dashboard as first tab (6 total) | Round 1 d3, revised round 2 d1 |
| Widget set | Stats + blocked items + initiative progress + recent activity | Round 1 d2 → B |
| Data refresh | Auto-poll 60s + manual refresh button | Round 1 d4 → B |
| Data sources | `GET /api/v1/overview` + `GET /api/v1/execution` (client-side composition) | Round 2 d4 (pending) |
| Data fetching | React Query `useQuery` (no new Zustand store) | Round 2 d2 (pending) |
| Layout | CSS Grid responsive 2-col desktop / 1-col mobile | Round 2 d3 (pending) |
| New backend endpoints | None — compose from existing APIs | Round 1 |
| Acceptance paths | `scenarios/swarm-manager/ui/src/**` | Round 1 d1/d5 → A |

## 8. Testing Plan

### Unit Tests (per widget)
- `SummaryCards`: renders correct counts from mock overview data; handles zero/empty states
- `BlockedItems`: renders blocked item list; shows empty state when none blocked; links navigate correctly
- `InitiativeProgress`: renders initiative names with rollup percentage; handles no-initiative state
- `RecentActivity`: renders execution list with status badges; handles empty execution history

### Integration Tests
- `DashboardPage`: mounts with mocked service responses; verifies all 4 widgets render; verifies loading state; verifies per-widget error isolation

### Route/Navigation Tests
- `/` loads `DashboardPage` (not redirect to `/backlog`)
- Dashboard tab is highlighted/active on `/` route
- Keyboard shortcut `1` navigates to Dashboard
- Existing tab routes (`/backlog`, `/scenarios`, etc.) still work

### Responsive Tests
- Mobile viewport (375px): single-column layout, all widgets stacked
- Desktop viewport (1280px): 2-column grid layout

## 9. Rollout / Validation Checklist

- [ ] `DashboardPage` renders with real data from overview API
- [ ] All stat cards show correct counts
- [ ] Blocked/unblocked items link to correct backlog detail pages
- [ ] Initiative progress reflects actual rollup data
- [ ] Recent activity shows execution runs with correct status badges
- [ ] "Dashboard" tab is active on `/` route
- [ ] Keyboard shortcut `1` opens Dashboard
- [ ] Existing tab routes still work (shortcuts shifted to 2-6)
- [ ] Auto-refresh works (data updates after 60 seconds)
- [ ] Manual refresh button triggers immediate data reload
- [ ] Mobile layout is usable (single-column, all widgets visible)
- [ ] Per-widget error handling works (one widget failing doesn't crash page)

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Overview API returns too much data for dashboard | Low | Medium | Use summary fields only; don't render full item list |
| 6 tabs crowd mobile bottom nav | Medium | Medium | Round 2 d1 addresses this — icon-only or scrollable nav |
| Execution endpoint may not support limit/pagination | Low | Low | Fetch all and slice client-side; defer pagination to follow-up |
| Two parallel API calls on mount may feel slow | Low | Low | Show loading skeletons per-widget; queries run in parallel |
| Keyboard shortcut renumbering (1→Dashboard) may confuse existing users | Low | Low | Brief transition; consistent with tab order |

## 11. Non-goals / Prohibited Patterns

- Do not add WebSocket/SSE infrastructure for v1
- Do not create new backend endpoints — compose from existing APIs
- Do not add user preferences/customization — keep it simple
- Do not duplicate data display that's already well-served by existing pages
- Do not merge Prompts and Settings pages — they serve different domains
- Do not add dependency graph visualization (follow-up feature)
- Do not create a new Zustand store for read-only dashboard data

## 12. Definition of Done

- Dashboard page loads at `/` showing overview data from existing API
- Navigation includes Dashboard as the first tab with keyboard shortcut
- All 4 widgets (stats, blocked, initiatives, activity) render correctly with real data
- Auto-refresh at 60s interval + manual refresh button
- Per-widget error isolation (partial failure tolerance)
- Tests pass for new components
- No regressions on existing pages
- Responsive on mobile viewports
- All lint/type/test errors fixed in modified files
- Scenario restarted and healthy
