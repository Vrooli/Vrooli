# Plan: Add Quick-Glance Execution Result Component

## Required Reading

- `prompt-manager skill read react-coherence` — component placement, state architecture, styling coherence
- `prompt-manager skill read ux` — information hierarchy, compact display, mobile responsiveness

## Problem Statement

When an execution finishes (completed, failed, needs_review, needs_fixup, canceled), users must navigate to the Executions page or execution details to see outcomes. There is no at-a-glance result indicator on backlog cards — the `AgentRunningBadge` only shows while the agent is active (statuses: pending, starting, running, needs_review) and disappears once the execution reaches a terminal or post-run state.

**Goal:** Show a compact result summary badge in the same visual slot as the `AgentRunningBadge` on backlog cards. Clicking it navigates to the execution details page. This gives users immediate outcome visibility without leaving the current page.

## Scope

### In Scope
- New `ExecutionResultBadge` component that renders for finished executions
- Displays in the same position as `AgentRunningBadge` on backlog cards
- Shows status (completed/failed/needs_fixup/canceled) with appropriate color coding
- Clickable — opens execution details page
- Optionally shows aggregate review classification if finalization completed (ready/needs_work/etc.)

### Out of Scope
- Changes to the execution data model or API
- Changes to the `AgentRunningBadge` itself (it already handles active states correctly)
- Changes to the execution details page
- Historical execution list on backlog cards (only show the most recent execution)

### Acceptance Criteria
- `acceptance_allow`: `scenarios/swarm-manager/ui/**`
- `acceptance_deny`: (none)

## Technical Context

### Current Architecture
- **`AgentRunningBadge`** (`src/components/backlog/agent-running-badge.tsx`): Shows a cyan pulsing badge when agent activity has an active status. Uses `useAgentActivitiesStore` with `selectLatestActivityForBacklog`.
- **`backlog-card.tsx`** (line 116): Renders `AgentRunningBadge` in the status bar row alongside status dot, status label, and scenario badge.
- **`useExecutionStore`** (`src/stores/execution-store.ts`): Holds all executions in `items[]`, sorted by createdAt desc. No selector for "latest execution for a backlog item" exists yet.
- **Status colors** defined in `src/types/constants.ts` lines 152-162 (e.g., completed=emerald, failed=red, canceled=amber, needs_fixup=orange).
- **Navigation**: `detail-selection-store.ts` has `selectExecution(executionId)` to open ExecutionDetailsPage.

### Data Flow
1. `ExecutionRecord` has `backlogKind` and `backlogName` fields — can match to backlog items
2. Execution store already has all executions cached and auto-refreshes every 6 seconds
3. Need a new selector: "latest finished execution for a given backlog kind+name"

## Approach

### Phase 1: Add Execution Selector
Add a `selectLatestFinishedExecutionForBacklog(state, backlogKind, backlogName)` selector to `execution-store.ts` that returns the most recent execution matching the backlog item with a terminal status.

Terminal statuses for this component: `completed`, `failed`, `needs_fixup`, `canceled`.

Note: `needs_review` is handled by `AgentRunningBadge` already (it's in its `ACTIVE_STATUSES` set), so we exclude it here to avoid showing both badges simultaneously.

### Phase 2: Create ExecutionResultBadge Component
New file: `src/components/backlog/execution-result-badge.tsx`

- Props: `backlogKind`, `backlogName`
- Uses the new selector from Phase 1
- Renders a compact pill badge styled per status:
  - **completed** (emerald): checkmark icon + "Completed" (or review classification if available)
  - **failed** (red): X icon + "Failed"
  - **needs_fixup** (orange): wrench icon + "Needs fixup"
  - **canceled** (amber): circle-slash icon + "Canceled"
- If finalization has `aggregateClassification`, show it as secondary info (e.g., "Completed · Ready" or "Completed · Needs work")
- `onClick`: calls `selectExecution(executionId)` to open details

### Phase 3: Integrate into Backlog Card
In `backlog-card.tsx`, render `ExecutionResultBadge` adjacent to `AgentRunningBadge`. The two are mutually exclusive by status — when the agent is active, only `AgentRunningBadge` shows; when finished, only `ExecutionResultBadge` shows.

### Phase 4: Tests
- Unit test for the new selector
- Component test for `ExecutionResultBadge` rendering different statuses
- Component test verifying click navigates to execution details

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Performance: selector runs on every backlog card render | Medium | Low | Memoize with `memo()`, selector filters by backlogKind+backlogName early |
| Badge overlap with AgentRunningBadge | Low | Medium | Statuses are mutually exclusive by design; add guard |
| Execution store not loaded when backlog page mounts | Low | Low | Badge simply returns null if no matching execution found |
| Stale execution data | Low | Low | Store auto-refreshes every 6s already |

## Implementation Sequence

1. Add selector to `execution-store.ts`
2. Create `execution-result-badge.tsx` component
3. Integrate into `backlog-card.tsx`
4. Write tests
5. Verify with `make test`

## Verification

- [ ] Badge appears on backlog cards for items with finished executions
- [ ] Correct color/icon for each terminal status
- [ ] Clicking badge opens execution details page
- [ ] Badge disappears when a new execution starts (AgentRunningBadge takes over)
- [ ] No badge shown for items with no executions
- [ ] Review classification shown when available
- [ ] No performance degradation with many backlog cards
