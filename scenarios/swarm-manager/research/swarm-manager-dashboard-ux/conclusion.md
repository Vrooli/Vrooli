# Research Conclusion: Swarm-Manager Dashboard Layout & Initiative Card Design

## Research Question
What should the swarm-manager dashboard page look like, and how should initiative cards, the needs-attention feed, settings relocation, and initiative schema extensions be designed to give the director team at-a-glance system health and initiative progress?

## Summary
The dashboard replaces the Settings tab with a three-section layout: snapshot stats bar, initiative card grid, and needs-attention feed. Settings moves behind a header gear icon into a modal dialog. The initiative model needs `priority` (int, 1-10) and `target_date` (optional string) fields. All dashboard data can be sourced client-side from existing API endpoints — no new backend endpoints are required for the MVP. A critical gap discovered during research: the UI has no service layer for the existing `/api/v1/initiatives` API, so an `initiative-service.ts` must be created as the first implementation step.

## Methodology
- Examined all existing UI components, services, types, and pages in `scenarios/swarm-manager/ui/src/`
- Read the full API handler and model code for initiatives (`api/internal/initiatives/`) and backlog (`api/internal/backlog/`)
- Reviewed the proto schema for Initiative and InitiativeRollup messages
- Analyzed the SettingsPage structure for dialog migration feasibility
- Verified data availability for each Needs Attention feed category against existing endpoints
- Conducted 2 workshop rounds with the user to settle key design decisions

## Findings

### Finding 1: Navigation & Routing
The current MainLayout has 5 tabs (Backlog, Scenarios, Execution, Prompts, Settings) defined as a config array. Routes are defined in App.tsx with a standard `<Route>` + `PageErrorBoundary` pattern. The BacklogPage already supports URL-synced filtering via `useUrlState` for `kind`, `status`, `sort`, and `finished` params — meaning the stats bar click-through to filtered backlog views is already partially supported. However, there is no `initiative` filter param, so filtering by initiative will need a small BacklogPage enhancement.

**Settled decisions:**
- Dashboard becomes the default landing page (`/` → `/dashboard`) (Round 1, d2=A)
- Settings moves to header gear icon → modal dialog (Round 1, d1=A)
- Tab order: Dashboard (1), Backlog (2), Scenarios (3), Execution (4), Prompts (5)

### Finding 2: Initiative API Exists but UI Has No Service Layer
The API has full CRUD + rollup at `/api/v1/initiatives` with endpoints for List, Create, Get, Update, Delete, AddItems, and RemoveItems. The List endpoint returns `InitiativeWithRollup` objects including computed rollup status. **However, the UI has zero references to these endpoints** — no `initiative-service.ts`, no `api-endpoints.ts` entry, nothing. The UI only knows about initiatives as a string field on backlog items. This is the biggest implementation gap: the dashboard cannot render initiative cards without a frontend service to call `GET /api/v1/initiatives`.

**Evidence:** `grep -r "initiatives" scenarios/swarm-manager/ui/src/` returns zero matches. The only `initiative` references in services are the string field on backlog items.

### Finding 3: Initiative Schema Needs Extension
The current `Initiative` model (in `api/internal/initiatives/model.go`) has: name, title, description, status, items, created, updated. Missing for the dashboard:
- `priority` (int, 1-10, default 5) — needed for card sorting
- `target_date` (optional string, ISO-8601) — needed for deadline display

The proto schema (`packages/proto/schemas/swarm-manager/v1/domain/backlog.proto`) mirrors this with fields 1-7. New fields would be `int32 priority = 8` and `string target_date = 9`. File-based storage means no migration — new fields default gracefully.

**Settled decision:** Use 1-10 priority scale, default 5 (Round 1, d5=A)

### Finding 4: Initiative Card Design
**Settled decisions from Round 2:**
- Stacked/segmented progress bar showing status distribution + numeric breakdown below (d1=A)
- Colored number badge for priority (red=1-3, amber=4-6, green=7-10) (d2=A)

**Card anatomy (top to bottom):**
1. Header: title (left) + colored priority badge (right) + optional target_date subtitle
2. Stacked progress bar: segments colored by status (green=completed, cyan=in_progress, red=failed, gray=pending) with fraction label (e.g., "7/12")
3. Status breakdown: compact text row (e.g., "3 ready · 2 running · 1 failed")
4. Next up: highest-priority unblocked item in ready status, or "All complete" / "All blocked"
5. Failed summary (conditional): "N failed" with most recent failure's item name
6. Quick actions: "View Items" → `/backlog?initiative=<name>`, "Run Next" → queues next unblocked item

**Unassigned pseudo-card:** Same layout but titled "Unassigned", no priority badge, sorted last (Round 1, d3=A). Hidden when unassigned count is 0.

**Grid layout:** 1 col (<640px), 2 cols (640-1024px), 3 cols (>1024px). Cards sorted by priority ascending, then completion % ascending.

### Finding 5: Snapshot Stats Bar
Single horizontal row of clickable stat cards. Data sourced from cached `useBacklogList()` query with client-side counting:

| Stat | Filter on Click |
|------|-----------------|
| Active Items (status ∉ {completed, archived}) | `?status=active` (custom handling) |
| In Progress | `?status=in_progress` |
| Failed | `?status=failed` |
| Ready to Go | `?status=ready` |
| Completed | `?status=completed` |
| Unassigned (no initiative) | `?unassigned=true` (new param) |

Desktop: single flex row. Mobile: 2×3 grid wrap. Each card: icon + label + count.

### Finding 6: Needs Attention Feed
Client-side composition from existing endpoints (Round 1, d4=B), 30-second refetch interval, no toast notifications (Round 2, d3=A).

**Data sources verified against existing API:**

| Category | Source | Endpoint/Query |
|----------|--------|----------------|
| Failed executions | `executionService.list({status: "failed"})` | `GET /api/v1/execution?status=failed` |
| Unresolved decisions | `backlogService.getPendingQuestions()` | `GET /api/v1/backlog/pending-questions` |
| Needs fixup | `backlogService.list()` filtered client-side by `status=needs_fixup` | Needs verification — `needs_fixup` isn't in `BacklogStatus` type |
| Pending feedback | `backlogService.getFeedbackSummary()` | `GET /api/v1/backlog/feedback-summary` |

**Gap identified:** The `needs_fixup` status is mentioned in the spec but is NOT in the `BacklogStatus` union type in `domain.ts`. The current statuses are: backlog, researching, ready, queued, in_progress, completed, failed, archived. Either `needs_fixup` needs to be added or this attention category should use `needs_review` (also not present) or be dropped from the MVP.

**Data contract:**
```typescript
interface AttentionItem {
  category: "failed_execution" | "unresolved_decision" | "needs_fixup" | "pending_feedback";
  itemKind: BacklogKind;
  itemName: string;
  detail: string;
  updatedAt: string;
  link: string;
}
```

Sort: most recent first. Display: latest 5, expandable. Empty state: "Nothing needs attention — all clear!"

### Finding 7: Settings Dialog Feasibility
SettingsPage is ~558 lines with 6 card sections, dirty-checking, navigation blocker, and per-section reset buttons. Converting to a dialog is feasible:
- Max-width 640px, max-height 80vh with internal scroll
- Navigation blocker becomes a close-confirm dialog
- Save + Close buttons in footer
- Mobile: nearly full-screen (95vw × 90vh)
- The existing card-based layout adapts well to narrower dialog width

### Finding 8: Implementation Decomposition
Three execute items in dependency order (Round 2, d4=A):
1. **Initiative schema + initiative service + settings dialog** — adds priority/target_date fields to model + proto, creates `initiative-service.ts`, builds settings dialog and gear icon
2. **Dashboard page** — stats bar, initiative cards, needs-attention feed (depends on #1 for initiative data)
3. **Tab/navigation changes** — replaces Settings tab with Dashboard, updates routing and keyboard shortcuts (depends on #2)

## Limitations
- **No `needs_fixup` status verified:** The spec mentions `needs_fixup` as an attention feed category, but this status doesn't exist in the current `BacklogStatus` type. This needs resolution before implementing the attention feed.
- **"Run Next" queue action complexity:** The quick action needs to identify the next unblocked ready item within an initiative and call the queue endpoint. This requires client-side dependency graph evaluation that may not exist in the current UI.
- **No initiative filter on BacklogPage:** The stats bar and card "View Items" action depend on filtering the backlog by initiative, which requires a new `initiative` URL param + filter logic on BacklogPage.
- **Rollup granularity:** `RollupStatus.pending` combines backlog + ready + researching + queued statuses into one number. The card status breakdown wants finer granularity (separate ready count). Either the rollup API needs enhancement or the UI must compute fine-grained counts from the full item list.
- **No real user testing:** These designs are based on technical analysis and workshop discussion, not user research or usability testing.

## Actions

### Action 1: Create backlog item — Add priority and target_date to initiative model
- **Kind**: execute
- **Title**: Add priority and target_date fields to initiative schema
- **Description**: Add `priority` (int32, 1-10, default 5) and `target_date` (optional string, ISO-8601) to the Initiative model in `api/internal/initiatives/model.go`, the proto schema, CreateRequest, UpdateRequest, and regenerate proto bindings. Also create `ui/src/services/initiative-service.ts` wrapping all `/api/v1/initiatives` endpoints, and build the settings modal dialog with gear icon trigger in the header.
- **Initiative**: swarm-manager-dashboard
- **Priority**: 1
- **Effort**: M
- **Acceptance allow**: `["scenarios/swarm-manager/**", "packages/proto/**"]`

### Action 2: Create backlog item — Implement dashboard page
- **Kind**: execute
- **Title**: Implement swarm-manager dashboard page with stats bar, initiative cards, and attention feed
- **Description**: Build the DashboardPage component with three sections: SnapshotStatsBar (6 clickable stat cards), InitiativeCardGrid (priority-sorted cards with stacked progress bars, status breakdown, next-up, quick actions, unassigned pseudo-card), and NeedsAttentionFeed (client-side aggregation from execution, pending questions, feedback summary endpoints). Use 30s react-query refetch. See research conclusion for full component specifications. Depends on initiative schema + service execute item.
- **Initiative**: swarm-manager-dashboard
- **Priority**: 1
- **Effort**: L
- **Depends on**: execute/swarm-manager-dashboard-schema-and-service
- **Acceptance allow**: `["scenarios/swarm-manager/ui/**"]`

### Action 3: Create backlog item — Update navigation for dashboard
- **Kind**: execute
- **Title**: Replace Settings tab with Dashboard tab and update routing
- **Description**: Replace the Settings tab with Dashboard in MainLayout tab config. Add Dashboard route in App.tsx. Change default route from `/backlog` to `/dashboard`. Add gear icon to header that opens SettingsDialog. Update keyboard shortcuts (1=Dashboard through 5=Prompts). Add `initiative` URL filter param to BacklogPage for card click-through. Depends on dashboard page existing.
- **Initiative**: swarm-manager-dashboard
- **Priority**: 1
- **Effort**: S
- **Depends on**: execute/swarm-manager-dashboard-page
- **Acceptance allow**: `["scenarios/swarm-manager/ui/**"]`
