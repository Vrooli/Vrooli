# Research Conclusion: Define the Swarm Manager Graph Workspace Contract

## Research Question
What should the canonical graph workspace architecture be for swarm-manager — covering the node/edge model, graph lenses, shell information architecture, and migration strategy — to replace the current tabbed CRUD shell with a relationship-first, graph-centered workspace?

## Summary
Swarm-manager should replace its 5-tab CRUD shell with a sidebar + graph canvas + inspector workspace built independently using React Flow + Dagre. Six entity types (Capture, BacklogItem, Initiative, Scenario, ExecutionRecord, Run) and 7 edge types form the canonical graph model. Three lenses (Topology, Flow, Operations) expose relationship, lifecycle, and operational views respectively. The sidebar follows an activity-first unified feed pattern. A dedicated `/api/v1/graph?lens=` endpoint serves lens-specific projections. Initiative-based clustering with status filters controls visual complexity. The Operations lens uses Server-Sent Events (SSE) for real-time push updates rather than polling. The workspace lives at `/graph?lens=X&select=kind/name`, with 301 redirects from legacy paths to preserve deep-link compatibility. The migration is a hard-cut replacement with no compatibility layer, decomposed into 6 execute items with clear dependencies.

## Methodology
1. **Codebase audit of swarm-manager UI**: 7 routes, ~154 TS files, 6 Zustand stores, 7 API services, 30+ domain types, 5-tab horizontal bar (desktop) / bottom nav (mobile), detail routes with immersive mobile mode
2. **Codebase audit of swarm-manager API**: 60+ HTTP endpoints, overview endpoint (items + initiatives + dep graph + summary), execution records with 10 statuses and parent chains, capture classification, agent-manager proxy (status/run-state/stop), depgraph package (cycle detection, topo sort, blocked/unblocked)
3. **Codebase audit of prompt-manager graph**: React Flow v12 + Dagre, 18 files (~2400 lines), Zustand store with highlight/dim/hide query modes, BFS neighborhood selection, desktop popover + mobile bottom sheet, performance optimizations (onlyRenderVisibleElements, lightweight edges >300 threshold, memo, useShallow)
4. **Store and domain type analysis**: 6 Zustand stores (all localStorage-persisted, 100-200 item caps), 5 relationship patterns already modeled (dependencies, initiative grouping, execution chains, archive links, agent run context)
5. **Cross-reference with orchestration summary**: carried forward 4 valid findings, discarded dashboard-page assumptions
6. **Migration impact analysis**: mapped all current routes, components, stores, and external deep links to their graph workspace equivalents

## Findings

### Finding 1: Current swarm-manager UX is a flat 5-tab CRUD shell
The UI has 5 top-level tabs (Backlog, Scenarios, Execution, Prompts, Settings) rendered as a horizontal tab bar on desktop (h-16 header) and bottom nav on mobile (h-16 fixed). Navigation is page-based with detail routes (`/backlog/:kind/:name`, `/scenarios/:name`). Detail routes trigger immersive mobile mode (bottom nav hidden, full-bleed content). Keyboard shortcuts 1-5 switch tabs; Ctrl+K relays to host switcher.

**Key observations:**
- Settings occupies a primary tab but is low-frequency
- Prompts are a top-level tab but should become drill-down/reference info
- No initiative-aware filtering or initiative service layer in the UI (backend API exists)
- Execution records are a flat list with no visual linkage to parent backlog items or scenarios
- Agent-run dropdown in header provides deep links: `navigate(/backlog/${run.backlogKind}/${run.backlogName})`

### Finding 2: Swarm-manager graph is fully independent — no prompt-manager dependency
Per user decision, swarm-manager builds its own graph with zero dependency on prompt-manager. Architectural patterns to adopt independently:
- React Flow v12 + Dagre rendering stack
- Zustand store for graph state: filters, layout mode, highlight/dim/hide query modes, viewport persistence via localStorage
- BFS neighborhood selection on node click (type-constrained traversal)
- Desktop: floating popover anchored to node position, clamped to viewport bounds
- Mobile: bottom sheet variant for inspector
- Performance: `onlyRenderVisibleElements`, lightweight straight edges above 300-edge threshold, `memo()` on node components, `useShallow()` for derived selectors
- MiniMap with node-count threshold (hide above ~120 nodes)
- Auto-fit with signature-based change detection
- Three layout modes: hierarchical (Dagre network-simplex), compact (Dagre tight-tree), grouped (custom lane-per-type)

### Finding 3: Canonical node/edge model maps to 6 entity types and 7+ edge types

**Node types:**
| Type | Source | Key Display Fields | Lens Visibility |
|------|--------|-------------------|-----------------|
| Capture | `captures/{id}/capture.json` | text, status, classification | Topology |
| BacklogItem | `{kind}/{name}/spec.json` | kind, title, status, priority, tags, initiative | Topology, Flow |
| Initiative | `.vrooli/initiatives/{name}.json` | title, status, items[], rollup counts | Topology |
| Scenario | filesystem + runtime | name, status (running/stopped/error) | Topology, Operations |
| ExecutionRecord | `.vrooli/execution-runs.json` | execution_id, backlog ref, status, mode, review_result | Flow, Operations |
| Run (agent) | agent-manager API | run_id, status, real-time state | Operations |

**Edge types:**
| Edge | From → To | Source | Lens |
|------|-----------|--------|------|
| depends_on | BacklogItem → BacklogItem | `depends_on` field | Topology |
| member_of | BacklogItem → Initiative | `initiative` field | Topology |
| classified_as | Capture → BacklogItem | `classification.items` | Topology |
| executes | ExecutionRecord → BacklogItem | `backlog_kind` + `backlog_name` | Flow |
| targets | BacklogItem → Scenario | `acceptance_allow` glob matching | Topology, Operations |
| follow_up | ExecutionRecord → ExecutionRecord | `parent_execution_id` | Flow |
| spawned_run | ExecutionRecord → Run | `run_id` | Operations |

### Finding 4: Overview endpoint and depgraph package provide strong graph data foundation
`GET /api/v1/overview` already returns all backlog items, all initiatives with rollup, a full dependency graph (edges as `[from, to]` pairs, plus unblocked/blocked classification), and summary stats. The `depgraph` package supports: edge enumeration, blocked/unblocked computation, dependents lookup, topological sort, and cycle detection.

**What a dedicated graph endpoint needs beyond overview:**
- Capture nodes and classified_as edges
- Execution records with executes/follow_up edges
- Agent run nodes with spawned_run edges and real-time status
- Scenario nodes with targets edges (resolved from acceptance_allow globs)
- Lens-specific filtering (only return nodes/edges relevant to the requested lens)

### Finding 5: Three lenses — each has a distinct purpose, node/edge subset, and default filter

**Topology Lens (default):**
- Shows: Captures, BacklogItems, Initiatives, Scenarios
- Edges: depends_on, member_of, classified_as, targets
- Purpose: Entity relationships, dependency chains, initiative grouping, scenario targeting
- Default filter: Hide completed/archived items, group by initiative (collapsed clusters with rollup counts), cap unclustered nodes at ~50

**Flow Lens:**
- Shows: BacklogItems, ExecutionRecords
- Edges: depends_on, executes, follow_up
- Purpose: Lifecycle progression from backlog → execution → review → completion
- Default filter: Show items in active statuses (queued, in_progress, needs_review, needs_fixup), hide completed chains

**Operations Lens:**
- Shows: Scenarios, ExecutionRecords, Runs
- Edges: targets, spawned_run, executes
- Purpose: Live operational state — what's running, what's healthy, what needs attention
- Default filter: Show only running/active entities, receive real-time updates via SSE

### Finding 6: Inspector action model per node type per lens

When a node is selected, the inspector opens as a desktop popover (anchored to node, clamped to viewport) or mobile bottom sheet. Each node type exposes a detail card plus lens-contextual actions:

| Node Type | Detail Card | Topology Actions | Flow Actions | Operations Actions |
|-----------|-------------|-----------------|-------------|-------------------|
| **Capture** | Text, attachments, classification status, suggested items | Classify, Create item from suggestion, Delete | — | — |
| **BacklogItem** | Kind, title, status, priority, tags, initiative, deps, readiness, workshop state | Edit, Queue, Workshop, Add dependency, Assign initiative, View files | Queue, View execution history, Follow-up | — |
| **Initiative** | Title, status, member items rollup (total/completed/in_progress/failed/pending) | Edit, Add/remove items, Archive | — | — |
| **Scenario** | Name, status, description, priority, tags, completeness | View files, Edit metadata | — | Start, Stop, Restart, View logs |
| **ExecutionRecord** | ID, backlog ref, status, mode, operation, review result, timing | — | View prompt trace, Follow-up, Retry, Trigger review, Cancel | View prompt trace, Stop run, View logs |
| **Run** | Run ID, status, started/finished, duration, error message | — | — | Stop, View in agent-manager |

Actions are contextual: a BacklogItem shows "Queue" and "Workshop" in Topology but "View execution history" in Flow. Nodes not visible in a given lens have no actions for that lens.

### Finding 7: Migration route mapping with new URL scheme

The hard-cut migration replaces all 7 current routes with a graph workspace under a new URL scheme. Legacy deep links are preserved via 301 redirects.

**New URL scheme:** The graph workspace lives at `/graph` with query parameters:
- `?lens=topology|flow|operations` — selects the active lens (defaults to `topology`)
- `&select=kind/name` — auto-selects a node and opens the inspector

**Route mapping:**
| Current Route | New Route | Migration Notes |
|---------------|-----------|-----------------|
| `/` → redirect to `/backlog` | `/graph` (Topology lens) | New default landing |
| `/backlog` | `/graph?lens=topology` | Backlog list becomes sidebar + graph nodes |
| `/backlog/:kind/:name` | 301 → `/graph?lens=topology&select=:kind/:name` | Auto-selects node, opens inspector |
| `/scenarios` | `/graph?lens=topology` (filtered to scenario nodes) | Scenario list becomes graph nodes |
| `/scenarios/:name` | 301 → `/graph?lens=topology&select=scenario/:name` | Auto-selects scenario node, opens inspector |
| `/execution` | `/graph?lens=flow` | Execution list becomes flow graph |
| `/prompts` | Settings/reference drawer (gear icon) | Demoted from primary nav |
| `/settings` | Settings modal/drawer (gear icon in header) | Demoted from primary nav |

**Agent-manager compatibility:** The agent-run dropdown currently navigates to `/backlog/:kind/:name`. These paths will 301-redirect to the graph workspace with the correct node selected. Agent-manager clients should eventually update to use the new `/graph?...` URLs directly, but the redirects provide immediate backward compatibility.

### Finding 8: Performance estimates and anti-chaos guardrails

**Estimated graph sizes** (based on store caps and typical usage):
- BacklogItems: 50-200 (store cap 200)
- Initiatives: 5-20
- Scenarios: 10-50
- Captures: 20-100 (store cap 100)
- ExecutionRecords: 50-200 (store cap 100 in store, more on server)
- Runs: 1-10 active at any time

**Total node count per lens:**
- Topology: 85-370 nodes (all captures + items + initiatives + scenarios)
- Flow: 100-400 nodes (items + executions)
- Operations: 11-60 nodes (scenarios + active executions + runs)

**Anti-chaos rules (initiative-based clustering + status filters):**
1. **Initiative clustering**: Items with an initiative are grouped into collapsed clusters showing rollup counts. Expand on click. Items without an initiative form an "Unassigned" group.
2. **Status filters**: Each lens has default status exclusions (see Finding 5). User can toggle to show all.
3. **Node cap**: If unclustered visible nodes exceed ~50, auto-collapse the lowest-priority items into a "More items" pseudo-node with count.
4. **Edge complexity**: Above 300 visible edges, switch to lightweight straight edges (no bezier curves). Above 500, suggest the user narrow filters.
5. **MiniMap**: Show when node count > 20, hide when > 120 (too dense to be useful).

**Operations lens real-time strategy — Server-Sent Events (SSE):**
The Operations lens uses SSE for push-based real-time updates rather than polling. The API opens an SSE stream that pushes scenario status changes, execution record state transitions, and agent run status updates as they occur. This provides lower latency than polling and eliminates wasted requests when nothing has changed. SSE infrastructure requirements:
- New SSE endpoint (e.g., `GET /api/v1/graph/stream?lens=operations`)
- Server-side event dispatch on status changes (scenario start/stop, execution state transitions, run status updates)
- Client-side EventSource with automatic reconnection and exponential backoff
- Connection lifecycle: open when Operations lens is active, close when user switches to another lens
- Stale-data fallback: if SSE connection drops, perform a single full fetch on reconnect before resuming the stream

### Finding 9: Mobile interaction model

The current mobile UI uses a bottom nav bar (h-16) with immersive mode for detail routes. The graph workspace uses an adaptive layout with simplified graph rendering:

**Recommended: Adaptive layout with simplified graph**
- **Portrait**: Sidebar collapses to a hamburger menu. Graph canvas is full-width. Pinch-to-zoom and pan enabled. Tap node → bottom sheet inspector (same pattern as prompt-manager mobile). Lens switcher in header.
- **Landscape**: Split view option — sidebar as narrow left panel (collapsible), graph canvas fills remaining width.
- **Touch optimizations**: Larger hit targets for nodes (min 44x44px), long-press for context menu (alternative to right-click), swipe-down to dismiss inspector bottom sheet.
- **Performance**: Reduce node detail on mobile (show only icon + title, hide stats). Disable MiniMap on mobile. Limit visible nodes to ~30 before suggesting filter narrowing.

This replaces the current bottom nav entirely. Navigation between lenses uses a header control (segmented control or dropdown), not a tab bar.

## Limitations
- **Node/edge taxonomy is proposed but unvalidated.** The 6 node types and 7 edge types are derived from data model analysis, not user validation. Edge cases (e.g., items targeting multiple scenarios, captures with failed classification) need real-world testing.
- **Performance thresholds are estimates.** The 300-edge threshold for lightweight edges and 120-node MiniMap threshold are adopted from prompt-manager. Actual swarm-manager graph density may require different tuning.
- **SSE infrastructure does not exist yet.** The API currently has no SSE support. Building the SSE endpoint, server-side event dispatch, and client reconnection logic adds implementation scope beyond a simple polling loop. If SSE proves too costly for v1, the execute item can fall back to polling as an interim step.
- **Inspector action model assumes current API capabilities.** Some actions (e.g., "View logs" for scenarios) may require new API endpoints or agent-manager integration not yet built.
- **Mobile graph interaction is inherently limited.** Complex graphs with many nodes are difficult to navigate on small screens regardless of optimizations. The ~30 node mobile cap may be too restrictive for power users.
- **Graph data endpoint does not exist yet.** The dedicated `/api/v1/graph?lens=` endpoint needs to be designed and built. The overview endpoint covers ~70% of Topology needs but nothing for Flow/Operations.
- **301 redirects require client-side router handling.** The new URL scheme (`/graph?lens=X&select=kind/name`) with 301 redirects from legacy paths requires that the SPA router intercepts old paths and transforms them. Agent-manager clients using programmatic navigation will need to be updated to use new URLs eventually.

## Actions

### Action 1: Create backlog item — Build graph data foundation and API endpoint
- **Kind**: execute
- **Title**: Swarm Manager Graph Data Foundation
- **Description**: Create the `/api/v1/graph?lens=topology|flow|operations` endpoint that serves lens-specific node/edge projections. Build on the existing overview endpoint and depgraph package. Add capture nodes, execution/follow-up edges, scenario-targeting edge resolution from acceptance_allow globs, and agent-run nodes. Return data in React Flow-compatible format (nodes array with type/position, edges array with source/target/type). Include server-side filtering by lens and default status exclusions. Add an SSE endpoint (`GET /api/v1/graph/stream?lens=operations`) for real-time push updates on the Operations lens — dispatching events on scenario status changes, execution state transitions, and agent run updates.
- **Initiative**: swarm-manager-graph-workspace
- **Priority**: 1
- **Effort**: M
- **Depends_on**: (this research item)

### Action 2: Create backlog item — Build graph workspace shell (sidebar + canvas + inspector)
- **Kind**: execute
- **Title**: Swarm Manager Graph Workspace Shell
- **Description**: Replace the 5-tab MainLayout with a sidebar + graph canvas + inspector shell at the new `/graph` route. Sidebar: activity-first unified feed with entity-type filter toggles. Graph canvas: React Flow v12 + Dagre, Zustand graph store with highlight/dim/hide query modes, viewport persistence. Inspector: desktop popover anchored to node, mobile bottom sheet. Header: title, lens switcher (segmented control), settings gear icon, agent-run status. Implement 301 redirects from legacy paths (`/backlog/:kind/:name` → `/graph?lens=topology&select=:kind/:name`, etc.). Remove the 5-tab navigation (Backlog, Scenarios, Execution, Prompts, Settings tabs).
- **Initiative**: swarm-manager-graph-workspace
- **Priority**: 1
- **Effort**: L
- **Depends_on**: (this research item)

### Action 3: Create backlog item — Implement Topology and scenario impact views
- **Kind**: execute
- **Title**: Swarm Manager Topology and Scenario Impact Views
- **Description**: Implement the Topology lens showing Captures, BacklogItems, Initiatives, and Scenarios with depends_on, member_of, classified_as, and targets edges. Initiative-based clustering with collapsed groups showing rollup counts. Default filter: hide completed/archived. Node cap at ~50 unclustered with "More items" overflow. Scenario impact: resolve acceptance_allow globs to show which items target which scenarios. Inspector actions per node type as defined in the research contract Finding 6.
- **Initiative**: swarm-manager-graph-workspace
- **Priority**: 1
- **Effort**: L
- **Depends_on**: graph-data-foundation, graph-workspace-shell

### Action 4: Create backlog item — Implement Flow and Operations views
- **Kind**: execute
- **Title**: Swarm Manager Flow and Operations Views
- **Description**: Implement the Flow lens (BacklogItems + ExecutionRecords with executes/follow_up edges, showing lifecycle progression) and Operations lens (Scenarios + ExecutionRecords + Runs with targets/spawned_run/executes edges). Operations lens connects to the SSE stream endpoint for real-time push updates — opening the EventSource when the lens is active and closing it on lens switch. Implement automatic reconnection with exponential backoff and a full-fetch fallback on reconnect. Default filters per lens as defined in the research contract. Inspector actions per node type per lens as defined in Finding 6.
- **Initiative**: swarm-manager-graph-workspace
- **Priority**: 2
- **Effort**: L
- **Depends_on**: graph-data-foundation, graph-workspace-shell

### Action 5: Create backlog item — Remove legacy tabbed shell and dead routes
- **Kind**: chore
- **Title**: Remove Legacy Tabbed Surfaces from Swarm Manager
- **Description**: Remove the old MainLayout tab bar, BacklogPage list view, ScenariosPage list view, ExecutionPage list view, and any dead routes/components after the graph workspace is live. Remove PromptsPage as a top-level route (content moves to reference drawer). Remove SettingsPage as a top-level route (content moves to settings modal). Clean up unused stores, services, and types. Verify 301 redirects from old paths work correctly. Update agent-manager deep links to use new `/graph?...` URLs directly.
- **Initiative**: swarm-manager-graph-workspace
- **Priority**: 3
- **Effort**: M
- **Depends_on**: topology-impact-views, flow-ops-views

### Action 6: Create backlog item — Mobile graph interaction optimization
- **Kind**: execute
- **Title**: Swarm Manager Mobile Graph Interaction
- **Description**: Optimize the graph workspace for mobile: adaptive layout (portrait full-canvas with hamburger sidebar, landscape split view), bottom sheet inspector, touch targets (min 44x44px), long-press context menu, reduced node detail (icon + title only), MiniMap disabled, ~30 node mobile cap with filter suggestions. Replace the current bottom nav bar entirely. Test on common mobile viewports.
- **Initiative**: swarm-manager-graph-workspace
- **Priority**: 2
- **Effort**: M
- **Depends_on**: graph-workspace-shell
