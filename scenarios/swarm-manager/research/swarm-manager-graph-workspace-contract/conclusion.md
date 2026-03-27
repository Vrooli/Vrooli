# Research Conclusion: Define the Swarm Manager Graph Workspace Contract

## Research Question
What should the canonical graph workspace architecture be for swarm-manager — covering the node/edge model, graph lenses, shell information architecture, and migration strategy — to replace the current tabbed CRUD shell with a relationship-first, graph-centered workspace?

## Summary
Swarm-manager should replace its 5-tab CRUD shell with a sidebar + graph canvas + inspector workspace. The graph will be built independently from prompt-manager (no shared dependency), using React Flow + Dagre as the rendering stack. Three lenses (Topology, Flow, Operations) will expose the 6 entity types and 7+ edge types already implicit in the data model. The sidebar follows an activity-first unified feed pattern with entity-type filters. The existing overview API endpoint and depgraph package provide a strong foundation for the graph data layer, though a dedicated graph endpoint will be needed to serve lens-specific projections.

## Methodology
1. **Codebase audit of swarm-manager UI**: routes (7 routes), pages, components (~154 TS files), stores (6 Zustand), services (7 API services), domain types (30+), navigation model (5-tab horizontal bar / mobile bottom nav)
2. **Codebase audit of swarm-manager API**: all HTTP endpoints (~60+), overview endpoint data shape, depgraph package algorithms, backlog/execution/capture/initiative/scenario handlers, protobuf contracts
3. **Codebase audit of prompt-manager graph**: React Flow v12 + Dagre stack, 18 graph files (~2400 lines total), store/service/component patterns, layout modes, selection/highlight/filter/query systems, responsive design, performance optimizations
4. **Cross-reference with orchestration summary**: carried forward 4 valid findings, discarded dashboard-page assumptions

## Findings

### Finding 1: Current swarm-manager UX is a flat 5-tab CRUD shell
The UI has 5 top-level tabs (Backlog, Scenarios, Execution, Prompts, Settings) rendered as a horizontal tab bar on desktop and bottom nav on mobile. Navigation is page-based with detail routes (`/backlog/:kind/:name`, `/scenarios/:name`). There is no sidebar, no graph, and no relationship visualization. The Backlog page is the default landing route. The UI comprises ~154 TypeScript files, 6 Zustand stores, 7 API services, and 30+ domain types.

**Key observations:**
- Settings occupies a primary tab but is low-frequency — confirmed by the archived dashboard research
- Prompts are a top-level tab but should become drill-down/reference info, not primary navigation
- No initiative-aware filtering or initiative service layer in the UI (backend initiative API exists but frontend has no corresponding store or dedicated page)
- Execution records are a flat list with no visual linkage to their parent backlog items or scenarios

### Finding 2: Prompt-manager graph is a strong pattern reference but swarm-manager will be fully independent
The prompt-manager uses React Flow v12 + Dagre for graph rendering with ~2400 lines across 18 files. Per user decision, swarm-manager will build its own graph implementation with zero dependency on prompt-manager code, allowing both to evolve independently.

**Architectural patterns to adopt independently:**
- React Flow + Dagre rendering stack (proven at scale in this codebase)
- Zustand store for graph state: filters, layout mode, highlight/dim/hide query modes, viewport persistence via localStorage
- BFS neighborhood selection on node click (type-constrained traversal)
- Desktop: floating popover anchored to node position, clamped to viewport bounds
- Mobile: bottom sheet variant for inspector
- Performance: `onlyRenderVisibleElements`, lightweight straight edges above 300-edge threshold, `memo()` on node components, `useShallow()` for derived selectors
- MiniMap with node-count threshold (hide above ~120 nodes)
- Auto-fit with signature-based change detection
- Three layout modes: hierarchical (Dagre network-simplex), compact (Dagre tight-tree), grouped (custom lane-per-type)

### Finding 3: Canonical node/edge model maps to 6 entity types and 7+ edge types
Six entity types are natural graph nodes with well-defined edges derivable from existing data:

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

**What the overview endpoint provides today:**
- `items[]` — all backlog items across all 5 kinds
- `initiatives[]` — all initiatives with rollup (total, completed, in_progress, failed, pending)
- `dependency_graph.edges` — `[from, to]` pairs for depends_on relationships
- `dependency_graph.unblocked` / `blocked` — items classified by dependency satisfaction
- `summary` — aggregate counts by status and kind

**What a graph endpoint would need beyond overview:**
- Capture nodes and classified_as edges
- Execution records with executes/follow_up edges
- Agent run nodes with spawned_run edges and real-time status
- Scenario nodes with targets edges (resolved from acceptance_allow globs)
- Lens-specific filtering (only return nodes/edges relevant to the requested lens)

### Finding 5: Three lenses confirmed — each has a distinct purpose and node/edge subset
Per user decision, all three lenses will be defined in the contract:

**Topology Lens (default):**
- Shows: Captures, BacklogItems, Initiatives, Scenarios
- Edges: depends_on, member_of, classified_as, targets
- Purpose: Entity relationships, dependency chains, initiative grouping, scenario targeting
- Default filter: Hide completed/archived items, group by initiative

**Flow Lens:**
- Shows: BacklogItems, ExecutionRecords
- Edges: depends_on, executes, follow_up
- Purpose: Lifecycle progression from backlog → execution → review → completion
- Default filter: Show items in active statuses (queued, in_progress, needs_review, needs_fixup), hide completed chains

**Operations Lens:**
- Shows: Scenarios, ExecutionRecords, Runs
- Edges: targets, spawned_run, executes
- Purpose: Live operational state — what's running, what's healthy, what needs attention
- Default filter: Show only running/active entities, auto-refresh for real-time data

### Finding 6: Archived dashboard research carried forward 4 applicable findings
From the orchestration summary:
1. No initiative service layer in frontend despite backend API existing
2. Backlog filtering needs initiative-aware URL/filter support
3. Overview aggregation is too narrow for a serious operations surface
4. Settings should not be a primary top-level navigation destination

Discarded: dashboard-page assumptions, initiative-cards-as-metaphor, page-first prompt visibility.

## Limitations
- **Node/edge taxonomy is proposed but unvalidated.** The 6 node types and 7 edge types are derived from data model analysis, not user validation.
- **Performance unknowns.** No measurement of real graph sizes in practice. Need to estimate: how many backlog items, executions, captures exist in a typical instance? Clustering and filter rules need empirical data.
- **Mobile interaction model not yet specified.** The current UI has responsive mobile nav, but graph interaction on mobile (pan/zoom/select on touch) is a distinct UX challenge.
- **Operations lens requires polling/real-time data.** Agent run status requires periodic refresh from agent-manager. The refresh interval and data staleness tolerance are unspecified.
- **Inspector action model not yet defined.** What actions are available when a node is selected in each lens? This is critical for the workspace to be operationally useful.
- **Graph data API shape not yet designed.** Whether to extend the overview endpoint or create a dedicated graph endpoint is unresolved.

## Actions
<!-- TBD — will be defined as research converges on remaining open questions (inspector actions, API shape, anti-chaos guardrails, migration plan) -->
