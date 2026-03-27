# Implementation Plan: Swarm-Manager Dashboard Layout & Initiative Card Design

## 1. Purpose

Design the new Dashboard page that replaces the Settings tab, featuring a snapshot stats bar, initiative-centric cards with rollup analytics, and a "Needs Attention" feed. Also design the settings relocation (gear icon → modal dialog) and initiative schema extensions.

## 2. Required Reading

```bash
prompt-manager skill read react-coherence ux implementation-plan-authoring
```

- `scenarios/swarm-manager/ui/src/components/layout/MainLayout.tsx` — current tab layout, header, mobile nav
- `scenarios/swarm-manager/ui/src/App.tsx` — route definitions, proxy basename handling
- `scenarios/swarm-manager/ui/src/pages/SettingsPage.tsx` — settings page to relocate into dialog
- `scenarios/swarm-manager/ui/src/types/domain.ts` — frontend Initiative/InitiativeRollup types, BacklogStatus
- `scenarios/swarm-manager/ui/src/components/backlog/initiative-badge.tsx` — current initiative display

## 3. Problem Statement

Initiatives are currently display-only blue badges on backlog cards. There is no dedicated view for initiative health, no rollup analytics, no filtering by initiative, and no way for the director team to quickly assess what needs attention. The Settings tab consumes a top-level navigation slot that would be better used for a Dashboard providing at-a-glance system health and initiative progress.

## 4. Scope

### In Scope
- Dashboard page layout specification (3 sections: stats bar, initiative cards, needs attention feed)
- Initiative card component specification (visual layout, data contract, interactions)
- Settings relocation to header gear icon → modal dialog
- Initiative schema changes (priority, target_date fields)
- Needs Attention feed data contract (sources, item shape, sort order)
- Unassigned items treatment (pseudo-card in initiative grid)
- Mobile responsive behavior for all new components
- Tab bar update (Dashboard replaces Settings, gear icon added to header)

### Out of Scope
- Backend API implementation for new endpoints
- Actual React component code (this is a research/design item)
- Backlog filtering enhancements beyond what stats bar click-through requires
- Initiative CRUD UI changes beyond what's needed for new fields
- Real-time WebSocket updates (polling via react-query is sufficient)

## 5. Current Technical Context

### Navigation Structure
- 5 tabs: Backlog, Scenarios, Execution, Prompts, Settings (MainLayout.tsx:22-28)
- Desktop: horizontal tab bar in header with keyboard shortcuts 1-5
- Mobile: fixed bottom navigation bar, hides on immersive detail routes
- Header right side: "Agents running" dropdown with active run badges, auto-refreshes every 5s
- Color scheme: slate-950 bg, slate-50 text, cyan-400 active accents

### Initiative Model (API)
```go
type Initiative struct {
    Name        string   `json:"name"`
    Title       string   `json:"title"`
    Description string   `json:"description,omitempty"`
    Status      string   `json:"status"` // active, completed, archived
    Items       []string `json:"items"`  // "kind/name" references
    Created     string   `json:"created"`
    Updated     string   `json:"updated"`
}

type RollupStatus struct {
    Total      int `json:"total"`
    Completed  int `json:"completed"`
    InProgress int `json:"in_progress"`
    Failed     int `json:"failed"`
    Pending    int `json:"pending"`
}
```

**Missing fields:** `priority` (int, 1-10, default 5), `target_date` (optional ISO-8601 date) — needed for card sorting and director team integration.

### Frontend Types
- `Initiative` and `InitiativeRollup` are proto-generated wrappers (domain.ts)
- `InitiativeWithRollup` pairs the two for list endpoints
- `InitiativeBadge` is a small inline pill (`bg-blue-500/15 text-blue-400`, 10px font, truncated at 20 chars)
- `BacklogStatus`: backlog | researching | ready | queued | in_progress | completed | failed | archived

### Settings Page
- 6 card sections: Theme, Execution Defaults, Workshop, Agent Behavior, UI Preferences, Save button
- Dirty checking, navigation blocker for unsaved changes
- Conditional field visibility (e.g., Max Auto Rounds depends on Auto-Advance toggle)
- Per-section reset buttons + global save

### Route Structure
- `/backlog`, `/scenarios`, `/execution`, `/prompts`, `/settings`
- Detail routes: `/backlog/:kind/:name`, `/scenarios/:name` (immersive on mobile)
- 404 catch-all
- Proxy-aware basename via `getProxyInfo()`

## 6. Target End State

### Navigation Changes
- **5 tabs** (reordered): **Dashboard** (new, slot 1), Backlog (slot 2), Scenarios (slot 3), Execution (slot 4), Prompts (slot 5)
- **Gear icon** in header (desktop: right of tab bar, before agents dropdown; mobile: in header bar) opens Settings modal dialog
- **Dashboard is the default route** (`/` redirects to `/dashboard`)
- Keyboard shortcuts renumbered: 1=Dashboard, 2=Backlog, 3=Scenarios, 4=Execution, 5=Prompts

### Dashboard Page — Three Sections

#### Section 1: Snapshot Stats Bar (top row)
A single horizontal row of clickable stat cards showing aggregate counts across all backlog items:

| Stat | Source | Filter on Click |
|------|--------|-----------------|
| Active Items | count where status ∉ {completed, archived} | `?status=active` |
| In Progress | count where status = in_progress | `?status=in_progress` |
| Failed | count where status = failed | `?status=failed` |
| Ready to Go | count where status = ready | `?status=ready` |
| Completed | count where status = completed | `?status=completed` |
| Unassigned | count where initiative is empty | `?unassigned=true` |

- Each card: icon + label + count, clickable → navigates to `/backlog?filter=<value>`
- Layout: horizontal flex, wraps on mobile (2×3 grid), single row on desktop
- Data source: existing backlog list endpoint with status aggregation (client-side count from cached backlog data)

#### Section 2: Initiative Cards (main area, scrollable grid)
Priority-sorted grid of initiative cards + one "Unassigned" pseudo-card.

**Sort order:** by priority ascending (1 = highest), then by completion % ascending (least complete first).

**Grid layout:** 1 column on mobile (<640px), 2 columns on tablet (640-1024px), 3 columns on desktop (>1024px).

**Card anatomy (top to bottom):**
1. **Header row:** Initiative title (left) + priority badge (right, colored 1-10)
2. **Progress bar:** Horizontal bar showing completed/total ratio, with fraction label (e.g., "7/12")
3. **Status distribution:** Row of colored dots/chips showing count per status:
   - ready (green), in_progress (cyan), failed (red), pending (gray)
4. **Next up:** "Next: [item title]" — highest-priority unblocked item in ready status. "All blocked" or "All complete" if none.
5. **Failed summary:** If failed > 0, show "N failed" in red with most recent failure's item name
6. **Quick actions row:** Two buttons:
   - "View Items" → `/backlog?initiative=<name>` (navigates to filtered backlog)
   - "Run Next" → queues the "Next up" item for execution (disabled if no ready item)

**Unassigned pseudo-card:** Same layout but titled "Unassigned", no priority badge, sorted last regardless of content.

**Target date display:** If initiative has `target_date`, show it as a subtle label below the title. If past due, highlight in red/amber.

#### Section 3: Needs Attention Feed (bottom, compact list)
Compact list of items needing human intervention, aggregated client-side from existing endpoints.

**Data sources (all client-side composition from existing queries):**

| Category | Icon | Source Query | Item Shape |
|----------|------|-------------|------------|
| Failed executions | ❌ | Execution runs with status=failed | item link + error summary + time |
| Unresolved decisions | ❓ | Workshop rounds with pending decisions | item link + decision count + round # |
| Needs fixup | 🔧 | Backlog items with status=needs_fixup | item link + fixup reason |
| Pending feedback | 💬 | Feedback summary with total_pending > 0 | item link + pending count |

**Sort:** Most recent first (by updated timestamp across all categories).
**Display:** Latest 5 items by default, "Show all" expands to full list. Each entry: category icon + item link + brief detail + relative time.
**Empty state:** "Nothing needs attention — all clear!" with a subtle checkmark.

### Settings Modal Dialog
- **Trigger:** Gear icon in header (desktop: positioned right of nav tabs, before agents dropdown; mobile: gear icon in top header bar)
- **Dialog:** Centered overlay, max-width 640px, max-height 80vh with internal scroll
- **Content:** All 6 current settings card sections, adapted for narrower dialog width
- **Behavior:** Navigation blocker on close with unsaved changes (confirm dialog), Save + Close buttons in footer
- **Mobile:** Dialog goes nearly full-screen (95vw × 90vh) with close button in top-right

## 7. Implementation Strategy

This is a **research/design** item — no code is produced. The deliverables are specifications detailed enough that a follow-up `execute` item can implement without further design input.

### Specification Deliverables (embedded in this plan)
1. **Dashboard wireframe specification** — Section 6 above covers layout, grid breakpoints, and component placement
2. **Initiative card component specification** — Section 6.2 defines card anatomy, data mapping, interactions
3. **Needs Attention feed data contract** — Section 6.3 defines sources, item shape, sort order, pagination
4. **Settings relocation design** — Section 6 (Settings Modal Dialog) defines trigger, dialog sizing, mobile behavior
5. **Initiative schema changes** — Section 8 defines new fields, defaults, proto updates

### Component Hierarchy (specification only)
```
DashboardPage
├── SnapshotStatsBar
│   └── StatCard (×6: Active, In Progress, Failed, Ready, Completed, Unassigned)
├── InitiativeCardGrid
│   ├── InitiativeCard (×N, sorted by priority then completion)
│   │   ├── CardHeader (title + priority badge)
│   │   ├── ProgressBar (completed/total)
│   │   ├── StatusDistribution (colored dots per status)
│   │   ├── NextUpLine (next ready item)
│   │   ├── FailedSummary (conditional)
│   │   └── QuickActions (View Items + Run Next)
│   └── UnassignedCard (pseudo-card, always last)
└── NeedsAttentionFeed
    └── AttentionItem (×N, sorted by recency)

SettingsDialog (in MainLayout, triggered by gear icon)
├── SettingsContent (reused from current SettingsPage internals)
└── DialogFooter (Save + Close)
```

### Data Flow (specification only)
```
react-query cache
├── useBacklogList() → stats bar counts + unassigned count
├── useInitiativesWithRollup() → initiative cards
├── useExecutionRuns({status: "failed"}) → needs attention (failed)
├── useWorkshopPending() → needs attention (unresolved decisions)
├── useBacklogList({status: "needs_fixup"}) → needs attention (fixup)
└── useFeedbackSummary() → needs attention (pending feedback)
```

## 8. Contract Decisions

### Initiative Schema Extensions
- Add `priority` field: int, 1-10 scale, default 5
- Add `target_date` field: string, optional, ISO-8601 date format
- Both fields added to Initiative model, CreateRequest, UpdateRequest
- Proto schema updated: `int32 priority = 8;` and `string target_date = 9;`
- **No migration needed** — file-based store, new fields default gracefully

### Status Terminology
- The rollup uses `failed` (not "blocked") — dashboard cards use "Failed" consistently
- "Blocked" is not a backlog status; dependency-based blocking is a separate graph computation
- If future items need dependency-blocked display, it would be a separate computed field on the rollup

### Stats Bar Click-Through Contract
- Each stat card click navigates to `/backlog?filter=<key>`
- Backlog page must accept filter query params (may need minor enhancement)
- Unassigned filter: `?unassigned=true` filters to items with no initiative

### Needs Attention Feed — No New Backend Endpoint
- All data sourced client-side from existing endpoints
- Future optimization: dedicated `/api/v1/dashboard/attention` endpoint can be added later
- Data contract for each attention item:
  ```typescript
  interface AttentionItem {
    category: "failed_execution" | "unresolved_decision" | "needs_fixup" | "pending_feedback";
    itemKind: BacklogKind;
    itemName: string;
    detail: string; // error summary, decision count, fixup reason, etc.
    updatedAt: string; // ISO-8601, for sort ordering
    link: string; // route path to the relevant page
  }
  ```

## 9. Testing Plan

Since this is a research/design item, the "testing" is **specification review validation**:

### Specification Completeness Checks
- [ ] Every dashboard section has defined: data source, layout, responsive breakpoints, empty state, interaction behavior
- [ ] Initiative card has defined: all visual elements, data mapping to existing API fields, edge cases (0 items, all complete, all failed)
- [ ] Needs Attention feed has defined: all categories, sort order, item shape, pagination, empty state
- [ ] Settings dialog has defined: trigger placement, dialog sizing, mobile behavior, unsaved changes handling
- [ ] Schema changes are backward-compatible (no migration required)

### Edge Case Coverage in Specifications
- [ ] Dashboard with 0 initiatives (empty state)
- [ ] Initiative with 0 items
- [ ] Initiative where all items are completed
- [ ] Initiative where all items are failed
- [ ] No items needing attention (empty feed)
- [ ] 20+ initiatives (scroll behavior, performance)
- [ ] Very long initiative title (truncation)
- [ ] Mobile layout at all 3 breakpoints

### Implementability Validation
- [ ] All data sources map to existing API endpoints (no new backend required for MVP)
- [ ] Component hierarchy follows react-coherence patterns
- [ ] No circular dependencies in data flow

## 10. Rollout/Validation Checklist

Since this is a research item, rollout = transitioning to an `execute` item:

- [ ] Plan reviewed and all workshop decisions are settled
- [ ] All readiness dimensions at 2+
- [ ] Create follow-up `execute` item: "Implement swarm-manager dashboard page"
- [ ] Create follow-up `execute` item: "Relocate settings to header dialog"
- [ ] Create follow-up `execute` item: "Add priority and target_date to initiative schema"
- [ ] Each execute item references this plan for specifications

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Settings dialog too complex for modal layout (6 sections, conditional fields) | Medium | Start with modal; if UX testing shows problems, can promote to slide-over or keep as route with gear icon navigation |
| Client-side Needs Attention aggregation too slow with many items | Low | Each source query is already cached by react-query; merging small arrays is trivial. Add dedicated endpoint later if needed |
| Initiative priority field breaks existing items | Low | Default to 5, file-based store — no migration needed |
| Dashboard becomes stale without frequent polling | Medium | Use react-query with refetchInterval (30s for stats, 60s for cards), same pattern as existing pages |
| Stats bar filter click-through requires backlog page changes | Low | Backlog page likely already supports query params; if not, minimal enhancement needed |
| Unassigned pseudo-card clutters grid when there are no unassigned items | Low | Hide pseudo-card when unassigned count is 0 |
| Tab reordering breaks user muscle memory for keyboard shortcuts | Medium | Shortcuts change (1=Dashboard instead of 1=Backlog), but consistent numbering left-to-right. One-time adjustment |

## 12. Non-goals / Prohibited Patterns

- Do not implement initiative CRUD on the dashboard — use existing backlog/initiative pages
- Do not add real-time WebSocket updates (polling via react-query is sufficient)
- Do not redesign existing backlog cards or list views
- Do not add initiative creation from dashboard (use existing flow)
- Do not create a separate "blocked" status — use existing status taxonomy
- Do not build a custom charting library — use simple HTML/CSS progress bars and colored dots
- Do not add drag-and-drop reordering of initiative cards

## 13. Definition of Done

- [ ] Dashboard wireframe specification complete with all 3 sections defined (stats bar, initiative cards, needs attention feed)
- [ ] Initiative card component specification with visual layout, data contract, all interactions, and edge cases
- [ ] Needs Attention feed data contract defined (source queries, item shape, sort order, pagination, empty state)
- [ ] Settings relocation design documented (gear icon placement, dialog layout, mobile behavior, unsaved changes)
- [ ] Initiative schema changes documented (new fields, defaults, backward compatibility, proto updates)
- [ ] Unassigned items treatment specified (pseudo-card behavior, visibility rules)
- [ ] Mobile responsive behavior specified for all new components at 3 breakpoints
- [ ] Component hierarchy and data flow documented
- [ ] All specifications are detailed enough for a developer to implement without further design input
- [ ] Edge cases documented for each component
