# Implementation Plan: Build Topology and Scenario Impact Views in the Graph Workspace

## 1. Purpose

Implement the Topology lens content layer on top of the graph workspace shell: initiative-based clustering with rollup counts, node capping for visual complexity management, scenario impact visualization via targets edges, dynamic MiniMap behavior, and lens-specific inspector actions for all four topology node types (Capture, BacklogItem, Initiative, Scenario).

## 2. Required Reading

```bash
prompt-manager skill read ux react-coherence test
```

Also read dependency plans and the research contract conclusion:
```bash
swarm-manager backlog file-get --kind execute --name swarm-manager-graph-data-foundation --path plan.md
swarm-manager backlog file-get --kind execute --name swarm-manager-graph-workspace-shell --path plan.md
swarm-manager backlog file-get --kind research --name swarm-manager-graph-workspace-contract --path conclusion.md
```

## 3. Problem Statement

The graph workspace shell (completed) renders topology lens nodes as flat, unclustered entities with no visual grouping, no node capping, and no inspector actions for any topology node type. The graph data foundation API (completed) returns topology data including initiative nodes, member_of edges, and targets edges — but initiative nodes lack rollup status counts needed for cluster summaries.

Specific gaps:
- **No initiative clustering**: All nodes render at the same level with no visual grouping by initiative. Users cannot see initiative progress at a glance.
- **No node capping**: Large backlogs render all ~200 items, overwhelming the canvas.
- **No edge type differentiation**: All edges render identically despite having different semantic meanings (depends_on vs member_of vs classified_as vs targets).
- **No topology inspector actions**: The action registry has an empty topology section — clicking a node in topology shows no actions.
- **MiniMap thresholds wrong**: Currently shows at >=15 nodes; spec requires show at >20, hide at >120.
- **Initiative rollup missing from API**: The API returns initiative name/title/status but not rollup counts (total/completed/in_progress/failed/pending), even though `InitiativeWithRollup` has this data available — the projection layer simply discards it at `projection.go:126-130`.

## 4. Scope

### In scope
- API: Add rollup counts to initiative node data in topology projection (~5 line change in `projection.go`)
- API: Add two thin convenience endpoints: `POST /captures/:id/create-item` and `POST/DELETE /initiatives/:name/members` (R2 d3=B, scoped in R3 d3=A)
- UI: Initiative-based visual clustering using React Flow sub-flows (parent-child nodes) with collapsed/expanded states and rollup count display
- UI: "Unassigned" group for items without an initiative
- UI: Node cap at ~50 unclustered visible nodes with "More items" pseudo-node
- UI: Edge type visual differentiation (color + dash pattern, no labels) with compact fixed legend in bottom-left corner
- UI: Dynamic MiniMap (show >20 nodes, hide >120)
- UI: Topology inspector actions for Capture, BacklogItem, Initiative, Scenario (full set, one pass)
- UI: Edge complexity management (straight edges >300, filter suggestion >500)
- Tests for all new behavior

### Out of scope
- Flow lens content (separate item: flow-ops-views)
- Operations lens content (separate item: flow-ops-views)
- Mobile-specific graph optimizations (separate item)
- WebSocket/SSE streaming (operations lens only)
- Legacy route cleanup (separate chore)

## 5. Current Technical Context

### Key files — API
| File | Role |
|---|---|
| `api/internal/graph/projection.go` | Topology projection builder — creates nodes/edges, filters by status. Initiative nodes built at lines 112-134 but rollup data discarded |
| `api/internal/graph/types.go` | Node (with `Data any`), Edge, GraphResponse structs |
| `api/internal/graph/interfaces.go` | InitiativeLister and other data source interfaces |
| `api/internal/initiatives/model.go` | Initiative, RollupStatus (total/completed/in_progress/failed/pending), InitiativeWithRollup structs |

### Key files — UI
| File | Role |
|---|---|
| `ui/src/surfaces/graph/components/GraphCanvas.tsx` | React Flow canvas with Dagre layout, MiniMap (currently threshold >=15) |
| `ui/src/surfaces/graph/components/GraphNode.tsx` | Custom node renderer — flat, no parent-child support |
| `ui/src/surfaces/graph/components/Inspector.tsx` | Floating inspector panel (desktop popover, mobile bottom sheet) |
| `ui/src/surfaces/graph/lib/action-registry.ts` | Inspector actions per lens x entity — topology section is **empty** (line 271: `topology: {}`) |
| `ui/src/surfaces/graph/lib/layout-utils.ts` | Dagre config for 3 layout modes |
| `ui/src/surfaces/graph/stores/graph-data-store.ts` | Nodes, edges, lens, entityFilters — no clustering logic |
| `ui/src/surfaces/graph/stores/graph-ui-store.ts` | Selection, highlight, layout, viewport |

### Existing patterns
- GraphNode renders all 6 entity types via a type->icon/color map (ENTITY_COLORS) — initiative is sky/Target icon
- Entity filter toggles in sidebar control graph visibility
- BFS neighborhood selection on node click
- Three layout modes: hierarchical, compact, grouped (all Dagre-based)
- Viewport persistence via localStorage
- Action registry uses factory functions returning action objects with handler/navigateTo
- Node Data field is untyped `any` — flexible for adding rollup fields

### API endpoints available for topology inspector actions
| Action | Endpoint | Status |
|---|---|---|
| Capture classify | `POST /captures/:id/classify` | Exists |
| Capture delete | `DELETE /captures/:id` | Exists |
| BacklogItem edit | `PATCH /backlog/:kind/:name` | Exists |
| BacklogItem queue | `POST /backlog/:kind/:name/queue` | Exists |
| BacklogItem view files | `GET /backlog/:kind/:name/files` | Exists |
| BacklogItem assign initiative | `PATCH /backlog/:kind/:name` (set initiative field) | Exists |
| BacklogItem add dependency | `PATCH /backlog/:kind/:name` (update depends_on array) | Exists (full array replace) |
| Scenario view files | `GET /scenarios/:name/files` | Exists |
| Scenario edit metadata | `PATCH /scenarios/:name` | Exists |
| Create item from capture | `POST /captures/:id/create-item` | Needs new endpoint |
| Initiative add/remove members | `POST/DELETE /initiatives/:name/members` | Needs new endpoint |

## 6. Target End State

### Initiative Clustering (R1 d1=A: React Flow sub-flows, R3 d1=A: default collapsed)
- Initiative nodes become React Flow parent nodes; member backlog items set `parentId`
- **All clusters collapsed by default** on initial topology load — users see initiative overview with rollup counts first, drill into clusters of interest
- Collapsed state: only the initiative cluster node visible with rollup counts (total/completed/in_progress/failed/pending)
- Click to expand -> shows member nodes as children inside the parent boundary
- Double-click selects the cluster for BFS neighborhood highlighting (cluster treated as unit)
- Items without an initiative go into an "Unassigned" group (also a parent node, collapsible)
- React Flow handles containment, dragging, and edge routing for parent-child relationships

### Layout Integration (R2 d2=A: two-pass layout)
- **Pass 1 (inter-cluster):** Run Dagre with only cluster-level nodes (initiative clusters + unassigned group + unclustered items) to determine cluster positions
- **Pass 2 (intra-cluster):** For each expanded cluster, run a sub-Dagre layout on its children to position them relative to the parent's origin
- Parent node dimensions computed from children's bounding box + padding

### Node Capping
- After clustering, if unclustered visible nodes exceed ~50, auto-collapse the lowest-priority items into a "More items (N)" pseudo-node
- Expanding the pseudo-node shows all items (user opt-in to complexity)

### Edge Visualization (R1 d2=A: color + dash pattern)
- depends_on: solid line, slate color
- member_of: dashed line, sky color (matches initiative node color)
- classified_as: dotted line, emerald color (matches capture node color)
- targets: solid line, violet color (matches scenario node color)
- Above 300 visible edges: switch to straight lines (no bezier)
- Above 500 visible edges: show filter suggestion banner

### Edge Legend (R2 d4=A: compact fixed bottom-left)
- Small floating panel in bottom-left corner (opposite MiniMap in bottom-right)
- 4 rows: colored line sample + edge type name
- Always visible when topology lens is active
- Collapsible via toggle

### MiniMap
- Show when visible node count > 20
- Hide when visible node count > 120

### Inspector Actions (R1 d3=A: full set in one pass)
- **Capture**: Classify, Create item from suggestion, Delete
- **BacklogItem**: Edit, Queue, Workshop, Add dependency, Assign initiative, View files
- **Initiative**: Edit, Add/remove items, Archive
- **Scenario**: View files, Edit metadata

### Convenience API Endpoints (R2 d3=B: thin endpoints, R3 d3=A: two endpoints only)
- `POST /captures/:id/create-item` — creates a backlog item pre-filled from capture text/tags, marks capture as classified
- `POST/DELETE /initiatives/:name/members` — add/remove backlog items from initiative

### Cluster Interaction (R1 d4=A: click-to-expand)
- Single click on collapsed cluster -> expands to show member nodes
- Double-click on cluster -> selects cluster as unit, BFS highlights all edges to/from any member
- Collapsed clusters aggregate edges: single aggregated edge per (source-cluster, target-node, edge-type) triple with a count badge showing the number of underlying edges (R3 d2=A)

## 7. Implementation Strategy

### Phase 1: API — Add rollup counts to initiative nodes
1. In `projection.go` `buildTopology()`, add rollup fields to initiative node Data map:
   ```go
   "rollup": map[string]any{
       "total":       iwr.Rollup.Total,
       "completed":   iwr.Rollup.Completed,
       "in_progress": iwr.Rollup.InProgress,
       "failed":      iwr.Rollup.Failed,
       "pending":     iwr.Rollup.Pending,
   },
   ```
2. Add test in `projection_test.go` verifying initiative nodes include rollup counts

### Phase 2: API — Thin convenience endpoints
1. `POST /captures/:id/create-item` handler:
   - Read capture by ID
   - Create backlog item with kind=execute, title from capture text (truncated), description from capture text, tags from capture tags
   - Mark capture as classified (status=classified)
   - Return created item
2. `POST /initiatives/:name/members` handler:
   - Accept `{ items: [{ kind, name }] }` body
   - Update each item's initiative field to the target initiative
   - Return updated items
3. `DELETE /initiatives/:name/members` handler:
   - Accept `{ items: [{ kind, name }] }` body
   - Clear each item's initiative field
   - Return updated items
4. Register routes and add tests

### Phase 3: UI — Clustering utilities + store changes (R3 d4=A: single clustering-utils.ts)
1. Create `lib/clustering-utils.ts` with pure functions:
   - `buildClusterHierarchy(nodes, edges, initiatives)` -> `{ clusters: ClusterGroup[], orphans: Node[] }`
   - `aggregateEdgesForCollapsed(edges, collapsedClusterIds)` -> `Edge[]` (single edge per type with count badge data)
   - `applyNodeCap(nodes, limit)` -> `{ visible: Node[], cappedCount: number }`
   - `buildUnassignedGroup(orphans)` -> `ClusterGroup`
2. Add `collapsedClusters: Set<string>` to graph-ui-store
3. Add `toggleClusterCollapse(clusterId)` and `setAllCollapsed(collapsed)` actions
4. Co-located test file: `clustering-utils.test.ts`

### Phase 4: UI — Edge type differentiation + legend
1. Define edge style map in `lib/edge-styles.ts`:
   - `depends_on`: `{ stroke: colors.slate[400], strokeDasharray: 'none' }`
   - `member_of`: `{ stroke: colors.sky[400], strokeDasharray: '6 3' }`
   - `classified_as`: `{ stroke: colors.emerald[400], strokeDasharray: '2 3' }`
   - `targets`: `{ stroke: colors.violet[400], strokeDasharray: 'none' }`
2. Custom edge component or apply styles via React Flow's `defaultEdgeOptions` + type-specific overrides
3. Edge complexity thresholds in GraphCanvas:
   - Above 300 edges: `edgeType: 'straight'`
   - Above 500 edges: render filter suggestion banner component
4. Create `EdgeLegend` component: compact floating panel, bottom-left, collapsible
5. Co-located tests for edge-styles and EdgeLegend

### Phase 5: UI — Initiative clustering with React Flow sub-flows
1. Transform API nodes into React Flow parent-child hierarchy using clustering-utils
2. Create `ClusterNode` component:
   - Collapsed: initiative title + rollup badge (total/completed/in_progress/failed/pending)
   - Expanded: initiative title header, children rendered inside parent boundary
3. Two-pass layout integration in layout-utils (R2 d2=A):
   - Pass 1: Dagre with cluster-level nodes only -> cluster positions
   - Pass 2: Per-expanded-cluster sub-Dagre -> child positions relative to parent
   - Compute parent dimensions from children bounding box + padding
4. **Initialize all clusters as collapsed by default** (R3 d1=A): populate `collapsedClusters` set with all initiative IDs on initial load
5. Wire collapse/expand: click handler toggles cluster state, triggers re-layout
6. Edge aggregation on collapse: use `aggregateEdgesForCollapsed()` to produce single aggregated edge per (source-cluster, target-node, edge-type) triple with **count badge** showing underlying edge count (R3 d2=A)
7. "Unassigned" group node for items without initiative

### Phase 6: UI — Node capping
1. After clustering, count unclustered visible nodes
2. If > ~50, sort by priority descending, collapse lowest-priority into "More items (N)" pseudo-node
3. Pseudo-node click expands to show all items
4. Track cap state in graph-ui-store

### Phase 7: UI — MiniMap + inspector actions
1. Update MiniMap threshold: show when `filteredNodes.length > 20`, hide when `> 120`
2. Populate topology section of action-registry.ts with all 4 node type action sets
3. Wire each action:
   - `classify`: `POST /captures/:id/classify`
   - `createItemFromCapture`: `POST /captures/:id/create-item` (new endpoint)
   - `delete` (capture): `DELETE /captures/:id`
   - `edit` (backlog): navigate to backlog detail page
   - `queue`: `POST /backlog/:kind/:name/queue`
   - `workshop`: navigate to workshop page
   - `addDependency`: open modal with entity picker, `PATCH /backlog/:kind/:name`
   - `assignInitiative`: open modal with initiative picker, `POST /initiatives/:name/members`
   - `viewFiles`: navigate to files page
   - `editInitiative`: navigate to initiative detail page
   - `addRemoveItems`: open modal with item picker, `POST/DELETE /initiatives/:name/members`
   - `archiveInitiative`: `PATCH /backlog` (update initiative status)
   - `viewScenarioFiles`: navigate to scenario files page
   - `editScenarioMetadata`: navigate to scenario detail page

### Phase 8: Polish + integration testing
1. End-to-end verification of all features
2. Responsive behavior check (clusters, inspector on mobile)
3. Performance test with ~200 nodes

## 8. Contract Decisions

### Initiative Clustering Approach (Workshop R1 d1 — settled: A)
Use React Flow v12's native parent-child node support. Initiative nodes become parent nodes; member backlog items are children with `parentId` set.

### Edge Differentiation Strategy (Workshop R1 d2 — settled: A)
Color + dash pattern, no labels. A legend in the canvas corner explains the mapping.

### Inspector Action Scope (Workshop R1 d3 — settled: A)
Full action set in one pass — all 15+ actions across all 4 topology node types.

### Cluster Interaction Model (Workshop R1 d4 — settled: A)
Single click expands collapsed cluster. Double-click selects cluster as unit for BFS.

### Testing Strategy (Workshop R2 d1 — settled: A)
Store-first + clustering utility tests. Pure functions for clustering transformations, unit tested. Component tests verify wiring, not React Flow internals.

### Layout Integration (Workshop R2 d2 — settled: A)
Two-pass layout: Dagre for inter-cluster positions, then relative sub-Dagre for intra-cluster child positions.

### API Endpoint Strategy (Workshop R2 d3 — settled: B)
Add thin convenience endpoints for multi-step operations: `POST /captures/:id/create-item` and `POST/DELETE /initiatives/:name/members`. All other inspector actions use existing endpoints.

### Edge Legend Placement (Workshop R2 d4 — settled: A)
Compact fixed legend in bottom-left corner, collapsible, always visible in topology lens.

### Default Cluster Collapse State (Workshop R3 d1 — settled: A)
All clusters collapsed by default on initial topology load. Users see initiative overview with rollup counts first, then drill into clusters of interest. Progressive disclosure: start zoomed out, let users navigate in.

### Aggregated Edge Treatment (Workshop R3 d2 — settled: A)
Single aggregated edge per (source-cluster, target-node, edge-type) triple with a count badge showing the number of underlying edges. Minimizes visual noise — cluster collapse is meant to simplify.

### Convenience Endpoint Scope (Workshop R3 d3 — settled: A)
Two new endpoints only: `POST /captures/:id/create-item` and `POST/DELETE /initiatives/:name/members`. All other actions use existing endpoints. Dependency addition uses existing PATCH with full array replace.

### Clustering Utility Module Structure (Workshop R3 d4 — settled: A)
Single `lib/clustering-utils.ts` with pure functions + `collapsedClusters: Set<string>` in graph-ui-store. No separate clustering store.

## 9. Testing Plan

### API Tests
| Test | File | What |
|---|---|---|
| Initiative rollup in projection | `projection_test.go` | Verify initiative nodes include rollup counts in Data map |
| Create-item-from-capture endpoint | `handler_test.go` | POST creates backlog item from capture, marks capture classified |
| Initiative member management | `handler_test.go` | POST/DELETE adds/removes items from initiative |

### UI Unit Tests (pure function testing)
| Test Module | File | What |
|---|---|---|
| Clustering hierarchy builder | `clustering-utils.test.ts` | `buildClusterHierarchy`: groups items by initiative, creates Unassigned group for orphans, handles empty initiatives |
| Edge aggregation | `clustering-utils.test.ts` | `aggregateEdgesForCollapsed`: merges edges to collapsed clusters, produces single edge per type with count badge, preserves edge types |
| Node capping | `clustering-utils.test.ts` | `applyNodeCap`: caps at threshold, sorts by priority, creates pseudo-node with correct count |
| Edge styles | `edge-styles.test.ts` | Each edge type maps to correct color/dash pattern; threshold logic for straight edges |
| Two-pass layout | `layout-utils.test.ts` (extend) | Inter-cluster layout produces non-overlapping cluster positions; intra-cluster children fit within parent bounds |

### UI Component Tests (wiring verification)
| Test | File | What |
|---|---|---|
| ClusterNode collapsed | `ClusterNode.test.tsx` | Renders initiative title + rollup badge when collapsed |
| ClusterNode expanded | `ClusterNode.test.tsx` | Shows children placeholder when expanded |
| EdgeLegend | `EdgeLegend.test.tsx` | Renders 4 edge types with correct colors; collapsible toggle works |
| Action registry topology | `action-registry.test.ts` (extend) | All 4 node types return correct action IDs; action handlers callable |
| Inspector topology actions | `Inspector.test.tsx` (extend) | Topology actions rendered for each node type |

### UI Store Tests
| Test | File | What |
|---|---|---|
| Collapse/expand state | `graph-ui-store.test.ts` (extend) | `toggleClusterCollapse` toggles set membership; `setAllCollapsed` bulk updates; initial state has all clusters collapsed |

## 10. Rollout / Validation Checklist

- [ ] `go build ./...` passes (API changes)
- [ ] `go test ./... -timeout 300s` passes (API changes)
- [ ] Initiative nodes in topology API response include rollup counts
- [ ] Convenience endpoints functional (create-item-from-capture, initiative members)
- [ ] Initiative clusters render collapsed with rollup summary
- [ ] Expanding a cluster shows member nodes inside parent boundary
- [ ] "Unassigned" group for items without initiative
- [ ] Node cap triggers at ~50 unclustered nodes with "More items" pseudo-node
- [ ] Edge types have distinct visual treatment (color + dash)
- [ ] Edge legend visible in bottom-left, collapsible
- [ ] Edge complexity: straight edges above 300, filter suggestion above 500
- [ ] MiniMap shows at >20 nodes, hides at >120
- [ ] All 4 topology inspector action sets functional
- [ ] All new UI components have co-located test files
- [ ] Clustering utility tests pass
- [ ] `gofumpt -l .` reports no formatting issues
- [ ] Mobile: clusters and inspector work on small screens

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| React Flow sub-flow parent-child may have layout quirks with Dagre | Dagre doesn't know about parent-child containment; positions may overlap | Two-pass layout (R2 d2=A): compute cluster-level layout first, then child positions relative to parent. Parent dimensions from children bounding box + padding |
| Cluster edge aggregation on collapse may lose edge semantics | User can't see which specific child has the dependency | Settled R3 d2=A: single aggregated edge per type with count badge; expand cluster to see individual edges |
| Large cluster expansion could overwhelm canvas | Visual overload | Animate expansion, maintain viewport focus on cluster; node cap applies to expanded unclustered items |
| Inspector action API calls may not all exist | "Create item from suggestion" needs server-side orchestration | Add thin convenience endpoints (R2 d3=B): `POST /captures/:id/create-item`, `POST/DELETE /initiatives/:name/members` |
| Node capping interacts with clustering | Priority ordering across clusters | Cap only unclustered nodes; clustered items are already managed |
| "Unassigned" group may be very large | Many items may lack initiative | Apply node cap to Unassigned group specifically |
| Two-pass layout performance with many expanded clusters | Re-layout on each expand/collapse | Memoize sub-Dagre results; only recompute affected cluster |
| Convenience endpoints may need transactional safety | create-item-from-capture is multi-step (create + classify) | Wrap in single handler — if item creation fails, capture stays unclassified |

## 12. Non-goals / Prohibited Patterns

- Do NOT build compound/nested subgraph rendering — use React Flow parent-child sub-flows
- Do NOT modify flow or operations lens behavior
- Do NOT add backward-compatibility shims — this is greenfield topology content
- Do NOT compute rollup client-side — the API already has the data, just expose it
- Do NOT add edge labels — decided against in R1 d2
- Do NOT add more than the two agreed convenience endpoints unless existing endpoints prove truly insufficient
- Do NOT test React Flow rendering internals — test our logic (clustering, actions, styles), not the framework

## 13. Definition of Done

- [ ] Initiative rollup counts included in topology API response
- [ ] Convenience endpoints: create-item-from-capture and initiative member management
- [ ] Initiative-based clustering with React Flow sub-flow parent-child nodes
- [ ] Collapsed clusters show rollup counts, expand on click
- [ ] All clusters collapsed by default on initial load
- [ ] "Unassigned" group for orphan items
- [ ] Node cap at ~50 unclustered visible nodes
- [ ] Edge type visual differentiation (color + dash) for all 4 topology edge types
- [ ] Aggregated edges on collapsed clusters show count badges
- [ ] Edge legend in bottom-left corner, collapsible
- [ ] Dynamic MiniMap (show >20, hide >120)
- [ ] All 4 topology inspector action sets implemented and functional
- [ ] Edge complexity management (straight >300, suggestion >500)
- [ ] Co-located tests for all new components and utilities
- [ ] API tests for rollup addition and convenience endpoints
- [ ] Go code formatted and linted
