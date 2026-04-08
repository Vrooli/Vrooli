# Fix: Topology View Not Rendering

## Required Reading
- `prompt-manager skill read scientific-debugging` — hypothesis-driven root cause analysis
- `prompt-manager skill read react-coherence` — React rendering patterns and state management

## Purpose
Fix the blank topology view in swarm-manager's graph workspace. Users see either an empty canvas or the "No nodes match the current graph controls" empty state when navigating to the topology lens, despite having active backlog items.

## Problem Statement
The topology ("topo") lens in the swarm-manager graph workspace renders blank. Users see either an empty canvas or the "No nodes match the current graph controls" empty state instead of the expected graph of backlog items, initiatives, captures, and scenarios. The user has **confirmed** active backlog items exist (round 1, d2:A), ruling out the "no data" scenario.

## Current Technical Context

### Data Pipeline
The topology view has a full end-to-end pipeline:
1. **API** (`projection_topology.go`): `buildTopology()` loads backlog items (excluding completed/archived), initiatives, captures, and scenarios. Encodes as protobuf JSON via `proto_response.go`.
2. **Frontend fetch** (`graph-data-store.ts:221`): `fetchGraph("topology")` calls `graphService.getGraph("topology")`.
3. **Proto parsing** (`graph-service.ts:319`): `parseProtoResponse(graphResponseSchema, data, "graph")` decodes the response, then `mapProtoNode()` maps each node to the internal `GraphNode` type using a switch on `data.value.case`.
4. **Presentation** (`graph-presentation.ts:178`): `buildGraphPresentation()` filters nodes by entity/status settings, then applies initiative grouping (clusters).
5. **Layout** (`GraphCanvas.tsx:145`): Dagre computes node positions from `processedNodes`/`processedEdges`.
6. **Rendering** (`GraphCanvas.tsx:212`): React Flow renders `styledNodes` and `styledEdges`.

### Key Files
- `scenarios/swarm-manager/api/internal/graph/projection_topology.go` — API topology builder
- `scenarios/swarm-manager/api/internal/graph/proto_response.go` — Protobuf encoding
- `scenarios/swarm-manager/ui/src/services/graph-service.ts` — Proto decoding + node mapping
- `scenarios/swarm-manager/ui/src/surfaces/graph/stores/graph-data-store.ts` — Data store + fetch logic
- `scenarios/swarm-manager/ui/src/surfaces/graph/lib/graph-presentation.ts` — Filtering + clustering
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/GraphCanvas.tsx` — React Flow canvas

### Node Type Mapping (Verified Consistent)
| Go `Node.Type` | Proto `case` | Frontend `entityType` | React Flow `type` |
|---|---|---|---|
| `BacklogItem` | `backlog` | `backlog` | `backlog` |
| `Initiative` | `initiative` | `initiative` | `initiative` |
| `Capture` | `capture` | `capture` | `capture` |
| `Scenario` | `scenario` | `scenario` | `scenario` |
| `ExecutionRecord` | `execution` | `execution` | `execution` |
| `AgentActivity` | `activity` | `agent-activity` | `agent-activity` |
| `Run` | `run` | `agent-run` | `agent-run` |

All 7 types are handled in `mapProtoNode()`, `encodeGraphNodeData()`, `NODE_TYPE_MAP`, and `ENTITY_REGISTRY`. No mismatches found.

## Root Cause Hypotheses

### H1: API returns empty node set — NEEDS VERIFICATION
The topology projection only includes non-completed, non-archived backlog items. If `LoadAll(nil)` returns items but all are completed/archived, nodes would be empty. The user confirmed active items exist (d2:A), but this needs API-level verification (d1:A).

### H2: Entity/status filters hiding all nodes — NEEDS VERIFICATION
`graph-settings-store.ts` persists per-lens entity and status filters to localStorage (`swarm-manager.graph.settings.v6`). Default settings enable all entity types. If a user previously toggled off all entity types or statuses, `filterGraphNodes()` would remove everything. The `DEFAULT_ENTITY_FILTERS` initializes all to `true`, but corrupted/migrated localStorage could override this.

### H3: Protobuf parsing/mapping failure — LOW PROBABILITY
Code review confirms all 7 node types are correctly handled in both encoding and decoding. The `default` case in `mapProtoNode()` logs a warning and creates a fallback node (not silent failure). Proto schema uses generated types from `@vrooli/proto-types`. No mismatch found between Go encoding and TS decoding.

### H4: Initiative clustering collapses all nodes into invisible clusters — POSSIBLE
When `groupingMode === "initiative"` (default for topology), `buildInitiativeTopologyPresentation()` clusters backlog items under initiative nodes. All clusters start collapsed. If cluster nodes render but child nodes are hidden (collapsed), the canvas shows only cluster nodes. If there are no unclustered items, only cluster nodes appear — this should still show something, not be blank. However, if all items belong to initiatives AND all clusters are collapsed, only cluster `GraphNode` entries exist. These have `type: "cluster"` which IS registered as a node type in React Flow. This hypothesis is less likely but worth testing.

### H5: React Flow container has zero height/width — LOW PROBABILITY
If the parent container has no dimensions, React Flow won't render. Unlikely given that other lenses work.

### H6: fetchGraph error is caught silently — NEEDS VERIFICATION
The `graph-data-store.ts` fetch path sets `error` on the store when the API call fails. `GraphCanvas.tsx:410-417` renders an error banner. If the error banner renders but is visually hidden (z-index, positioning), the user might perceive a blank canvas. Also check if the error state is being cleared immediately after being set.

## Diagnostic Strategy (API-first — per d1:A)

1. **Check API response** — `curl localhost:<port>/api/v1/graph?lens=topology` — verify node/edge counts in `meta`.
2. **Check browser console** — Look for proto mapping warnings, React errors, or fetch failures.
3. **Inspect store state** — Check `useGraphDataStore.getState()` in browser console for `nodes`, `edges`, `error`, `loading` state.
4. **Check localStorage filters** — Inspect `swarm-manager.graph.settings.v6` for entity filter state.
5. **Toggle grouping mode** — Switch from "initiative" to "none" in graph controls to test H4.

## Target End State
- Topology lens renders backlog items, initiatives, captures, and scenarios as a connected graph.
- Empty-state detection logs pipeline-stage node counts for future debugging (per d3:B).
- No regression in other lenses (focus, operations).

## Scope

### acceptance_allow
- `scenarios/swarm-manager/**`

### Likely Change Targets
- **If H1/H2:** Fix filter logic or add safety reset for corrupted localStorage state.
- **If H3:** Fix proto mapping (unlikely based on code review).
- **If H4:** Fix clustering visibility or default grouping mode.
- **If H6:** Fix error display or surface errors more clearly.
- **Defensive (d3:B):** Add pipeline-stage node count logging to `buildGraphPresentation()` and `fetchGraph()`.

## Implementation Strategy

<!-- TBD — depends on root cause identification from diagnostics -->

### Phase 1: Diagnosis
Run the API-first diagnostic strategy above to identify which hypothesis is correct.

### Phase 2: Fix
Apply targeted fix based on confirmed root cause (per d3:A — minimal targeted fix).

### Phase 3: Defensive Improvement (per d3:B)
Add diagnostic logging at each pipeline stage:
- `fetchGraph()`: Log API response node/edge count
- `buildGraphPresentation()`: Log pre-filter and post-filter node counts
- `filterGraphNodes()`: Log which entity types were filtered out

## Test Plan
- Verify topology lens shows nodes after fix.
- Verify other lenses (focus, operations) still work.
- If filter-related: verify filters can be toggled without hiding all content unexpectedly.
- Run existing graph tests: `cd scenarios/swarm-manager/ui && npx vitest run --reporter=verbose -- graph-presentation graph-data-store GraphCanvas`

## Risks + Mitigations
| Risk | Impact | Mitigation |
|---|---|---|
| localStorage corruption from settings migration | Users stuck with broken filters | Add version check + fallback to defaults on parse failure |
| Proto schema drift | Silent node mapping failures | The `default` case already logs warnings — add metric/count |
| Capture edge key mismatch | `ci.Title` used as name in `backlogItemKey()` (projection_topology.go:137) — edges may not connect | Verify capture classification item fields; fix if `Title` should be `Name` |

## Non-goals / Prohibited Patterns
- Do not refactor the entire graph pipeline — scope to the specific broken layer.
- Do not change the protobuf schema or regenerate types unless proto mapping is confirmed root cause.
- Do not add new UI features beyond the fix + diagnostics.

## Contract Decisions
- **Diagnostic approach:** API-first (round 1, d1:A)
- **Data confirmation:** Active backlog items exist; this IS a bug (round 1, d2:A)
- **Fix strategy:** Minimal fix + defensive diagnostics at pipeline stages (round 1, d3:B)

## Rollout / Validation Checklist
- [ ] Identify root cause via API-first diagnostics
- [ ] Implement targeted fix
- [ ] Add pipeline-stage diagnostic logging
- [ ] Run existing graph tests
- [ ] Manual verification: topology lens shows nodes
- [ ] Manual verification: focus and operations lenses unaffected
- [ ] `vrooli scenario restart swarm-manager`

## Definition of Done
- Topology lens renders backlog items, initiatives, and their relationships.
- Pipeline-stage node counts are logged for future debugging.
- All existing graph tests pass.
- No regression in focus or operations lenses.
