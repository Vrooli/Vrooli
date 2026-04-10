# Plan: Show Running Agent Activities in Swarm-Manager Ops Tab

## Required Reading
- `prompt-manager skill read implementation-plan-authoring` — plan structure and quality gates
- `prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement` — canonical development patterns

## Problem Statement
The operations graph lens (`buildOperations()` in `projection.go`) currently only includes backlog items, execution records, and initiatives. Agent activities — tracked in `agentactivity.Record` and already wired into the projection config via `p.activity` — are excluded from this lens. This means the operations graph provides an incomplete view of active agent work: you can see executions running, but not the individual agent activities driving them.

**User clarification (round 1, d1):** The "ops tab" refers to the **operations graph lens** (a graph projection rendered by GraphCanvas), not the ExecutionsTab sidebar list. Activities should appear as **graph nodes** in the operations lens, not as list items.

## Goal
Add agent activity nodes and their associated run nodes to the operations graph lens, giving operators a complete view of all active agent work — executions *and* the activities powering them.

## Scope

### In Scope
- Modify `buildOperations()` in `api/internal/graph/projection.go` to load and include active agent activity nodes
- Reuse existing `addActivityNodesAndEdges()` helper directly (settled: round 2, d3)
- Add activity nodes with edges: `activity_for` (activity→owner), `records_activity` (execution→activity), `spawned_run`/`continued_run` (activity→run)
- Include run nodes when activities have a RunID (settled: round 2, d2)
- Only include activities whose owner is already in the operations graph (settled: round 2, d1)
- Filter activities to active statuses only: pending, starting, running, needs_review (settled: round 1, d2)
- Update tests in `api/internal/graph/projection_test.go` to expect activity nodes

### Out of Scope
- Changing the activity polling interval or mechanism
- Modifying the ActivityTab sidebar feed
- Adding new API endpoints (all data is already available via `p.activity`)
- Frontend changes to GraphCanvas (it already renders AgentActivity nodes in the flow lens)
- Changes to the ExecutionsTab sidebar list
- Including activities whose owner is not in the operations graph (standalone/disconnected nodes)

### Acceptance Patterns
- `acceptance_allow`: `scenarios/swarm-manager/**`
- `acceptance_deny`: none

## Technical Context

### Current Architecture
- **`buildOperations()`** (`projection.go:886`): Builds operations lens with backlog items (actionable statuses), execution records (active statuses), and initiatives. Currently excludes activities.
- **`addActivityNodesAndEdges()`** (`projection.go:543`): Existing helper used by `buildFlowForBacklogItem()`. Creates activity nodes and edges (`activity_for`, `records_activity`, `spawned_run`/`continued_run`). Accepts `executionIDs` and `ownerNodeIDs` maps to determine which edges to create — edges are only created when the target node exists. This makes it safe to reuse directly in the operations context.
- **`buildActivityNode()`** (`projection.go:421`): Creates a single activity graph node.
- **`p.activity`**: The `agentactivity.Service` is already wired into the projection config (`main.go:551-598`).
- **`GraphAgentActivityNodeData`** (`types.go:84-98`): Node data structure already exists.
- **Active statuses** (`agentactivity/types.go:41-44`): `StatusPending`, `StatusStarting`, `StatusRunning`, `StatusNeedsReview`.
- **Frontend**: GraphCanvas already renders AgentActivity nodes (used in flow lens). No frontend changes expected.

### Key Functions
- `buildOperations()` — main target for modification
- `addActivityNodesAndEdges()` — reusable helper, will be called directly with all active activities
- `buildActivityNode()` — creates a single activity graph node (called internally by the helper)
- `isActiveStatus()` (`agentactivity/types.go:143`) — checks if a status is active

### Test Impact
- `projection_test.go:331` has an explicit assertion in `TestProjectOperations`: `nodeTypes["AgentActivity"] != 0` → error. This must be updated to expect activity nodes when activities exist.

## Target End State
The operations graph lens includes active agent activity nodes connected to their owning backlog items and executions. Run nodes appear when activities have an associated RunID. The graph provides a complete picture of all active agent work scoped to actionable entities already in the operations view.

## Implementation Strategy

### Phase 1: Add Activities to Operations Graph

**Step 1: Load active activities in `buildOperations()`**
- After loading executions (around line 900), query `p.activity` for activities filtered to active statuses (`StatusPending`, `StatusStarting`, `StatusRunning`, `StatusNeedsReview`)
- Use the same query pattern as `buildFlowForBacklogItem()` for loading activities

**Step 2: Build owner and execution ID maps**
- Collect the set of node IDs already in the graph (backlog items, executions, initiatives) into `ownerNodeIDs` and `executionIDs` maps
- These maps are passed to `addActivityNodesAndEdges()` to control edge creation — only edges whose target exists in the graph are created
- This naturally enforces the "only include activities whose owner is already in the graph" decision: activities without a matching owner node will have no `activity_for` edge, and `addActivityNodesAndEdges()` will still create the activity node but it won't be connected. If we want to fully exclude such activities, filter them before passing to the helper.

**Step 3: Filter to owned activities only**
- Before calling `addActivityNodesAndEdges()`, filter the activity list to only those whose owner ID matches a node already in the graph (from `ownerNodeIDs`)
- This ensures no disconnected activity nodes appear in the operations lens

**Step 4: Add activity nodes and edges via helper**
- Call `addActivityNodesAndEdges(nodes, edges, filteredActivities, executionIDs, ownerNodeIDs)`
- The helper handles: activity node creation, `activity_for` edges, `records_activity` edges, run node creation (when RunID is set), and `spawned_run`/`continued_run` edges

### Phase 2: Update Tests

**Step 1: Update `TestProjectOperations` in `projection_test.go`**
- Seed active agent activities linked to existing executions/backlog items in the test
- Update the assertion at line 331 to expect AgentActivity nodes (instead of asserting they're absent)
- Assert correct edge types are created (`activity_for`, `records_activity`)
- Assert run nodes appear when activities have RunIDs

**Step 2: Add negative test cases**
- Verify terminal activities (complete, failed, cancelled) are excluded
- Verify activities whose owner is not in the operations graph are excluded
- Verify existing operations lens behavior (backlog items, executions, initiatives) is unchanged

## Contract Decisions
- Activity nodes use the existing `GraphAgentActivityNodeData` structure (no new types)
- Edge types reuse existing `activity_for`, `records_activity`, `spawned_run`/`continued_run` (no new edge types)
- No new API endpoints or query parameters
- No frontend contract changes (GraphCanvas already handles AgentActivity nodes)

## Testing Plan
- **Unit tests**: Modify `TestProjectOperations` in `projection_test.go`:
  - Seed active agent activities linked to existing executions/backlog items
  - Assert AgentActivity nodes are present with correct data
  - Assert edges (`activity_for`, `records_activity`) are created
  - Assert run nodes appear when activities have RunIDs
  - Assert terminal activities are excluded
  - Assert activities without owners in the graph are excluded
- **Manual verification**: Switch to operations lens in UI, confirm activity nodes appear and connect correctly

## Rollout/Validation Checklist
- [ ] `go build ./...` passes in `scenarios/swarm-manager/api`
- [ ] `go test ./...` passes in `scenarios/swarm-manager/api`
- [ ] Operations lens shows activity nodes in UI when activities are running
- [ ] Activity nodes connect to their owning backlog items and executions
- [ ] Run nodes appear when activities have RunIDs
- [ ] Terminal activities do not appear
- [ ] Existing operations lens behavior unchanged

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Activity nodes clutter operations graph | Medium | Low | Only active statuses shown; typically few concurrent activities |
| `addActivityNodesAndEdges()` assumptions don't hold for operations context | Low | Medium | Verified: helper uses `ownerNodeIDs`/`executionIDs` maps to control edges — safe for operations context |
| Edge targets missing (activity references execution/owner not in graph) | N/A | N/A | Eliminated by pre-filtering activities to those with owners in the graph |
| Frontend rendering issues with activity nodes in operations lens | Low | Low | Already renders them in flow lens; same node type |

## Non-goals / Prohibited Patterns
- Do not add standalone/disconnected activity nodes (activities without an owner in the graph)
- Do not modify the ExecutionsTab sidebar list
- Do not add new API endpoints
- Do not change the activity polling interval
- Do not add compatibility shims — this is a greenfield addition to existing projection logic

## Definition of Done
- Operations lens graph includes active agent activity nodes (pending, starting, running, needs_review)
- Activities are connected to their owning backlog items via `activity_for` edges
- Activities are connected to their executions via `records_activity` edges
- Run nodes appear when activities have a RunID, connected via `spawned_run`/`continued_run` edges
- Only activities whose owner is in the operations graph are included
- Terminal activities (complete, failed, cancelled) are excluded
- Existing operations lens behavior (backlog items, executions, initiatives) unchanged
- Tests updated and passing
- `go build` and `go test` pass
