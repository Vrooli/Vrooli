# Plan: Show Running Agent Activities in Swarm-Manager Ops Tab

## Required Reading
- `prompt-manager skill read react-coherence` — state architecture, sharing decision tree, component organization
- `prompt-manager skill read implementation-plan-authoring` — plan structure and quality gates

## Problem Statement
The Ops tab (`ExecutionsTab.tsx`) currently only displays execution records from `useExecutionStore`. Agent activities — tracked separately in `useAgentActivitiesStore` — are not shown in this tab, even when they represent actively running agents. This means the ops tab provides an incomplete view of all active agent work.

## Goal
Surface running (and recently completed) agent activities alongside executions in the Ops tab, giving operators a unified view of all active agent work.

## Scope

### In Scope
- Display agent activities in the Ops tab alongside executions
- Reuse existing `useAgentActivitiesStore` data (already polled every 5s by GraphWorkspace)
- Activity items should show: owner title, purpose, status badge, relative timestamp
- Filtering and sorting should work across both data types

### Out of Scope
- Changing the activity polling interval or mechanism
- Backend API changes (all data is already available)
- Modifying the ActivityTab or its feed logic
- Adding new activity detail views (clicking an activity navigates via existing mechanisms)

## Technical Context

### Current Architecture
- **ExecutionsTab** (`ui/src/surfaces/graph/components/sidebar/ExecutionsTab.tsx`): Renders executions from `useExecutionStore`
- **useAgentActivitiesStore** (`ui/src/stores/agent-activities-store.ts`): Zustand store with activities, already polled by `GraphWorkspace` every 5s
- **Activity statuses**: `pending | starting | running | needs_review | complete | failed | cancelled | unspecified`
- **Execution statuses**: `pending | starting | running | needs_review | validating | needs_fixup | completed | failed | canceled`
- Both have similar status color semantics

### Key Types
- `AgentActivity` from `ui/src/types/domain.ts` — activityId, ownerType, ownerKind, ownerName, purpose, status, timestamps, etc.
- `ExecutionRecord` from `ui/src/types/domain.ts` — executionId, backlogKind, backlogName, status, mode, timestamps, etc.

## Implementation Approach

<!-- TBD — pending decision on unified vs sectioned display -->

## Phases

### Phase 1: Core Integration
<!-- TBD — depends on display approach decision -->

## Testing Strategy
<!-- TBD -->

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Activity items clutter the ops view | Medium | Low | Filter to active-only by default, with toggle |
| Confusion between activity and execution items | Medium | Medium | Visual differentiation (icon, label, or subtle styling) |
| Performance with frequent activity polling | Low | Low | Already handled — store is polled independently |

## Acceptance Criteria
- Ops tab shows running agent activities alongside executions
- Activities are visually distinguishable from executions
- Existing execution filtering/sorting continues to work
- Activity filtering integrates with or extends existing filters
- Empty state still works when no executions or activities exist
