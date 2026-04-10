# Meta-Orchestrator Summary

## Source
Planning session covering a graph-first rewrite of the swarm-manager UX.

The original direction was an initiative-centric dashboard that replaced the Settings tab. During planning, that expanded into a broader conclusion: swarm-manager should become a relationship-first workspace with a sidebar-driven shell and graph-centered main view.

## Obsolete Line Of Work Archived
The following backlog line was archived because it no longer reflects the intended end state:
- `idea/dashboard`
- `research/swarm-manager-dashboard-ux`
- `execute/swarm-manager-initiative-enhancements`
- `execute/swarm-manager-dashboard-page`
- `execute/swarm-manager-initiative-management-ui`
- initiative `swarm-manager-dashboard`

## Reusable Signal Extracted From Archived Dashboard Research
Keep these findings where they still apply:
- The frontend currently has no initiative service layer despite the backend initiative API already existing.
- Backlog filtering needs stronger initiative-aware URL/filter support.
- Existing overview aggregation is too narrow for a serious operations surface.
- Settings should not remain a primary top-level navigation destination.

Discard these assumptions from the old plan:
- a dashboard page should be the primary new surface
- initiative cards are the main organizing metaphor
- prompt visibility should stay page-first rather than step-first

## Decisions Made
- Swarm Manager should move from a tabbed CRUD-like shell to a sidebar + graph workspace.
- The graph should be the center of gravity, but it should support multiple lenses rather than one giant everything-graph.
- At minimum the graph model should account for captures, backlog items, initiatives, scenarios, and runs.
- The product should expose both relationship topology and lifecycle/flow understanding.
- Prompts should become drill-down/reference information attached to flow steps, not a primary navigation concept.
- The migration should be hard-cut and greenfield. Do not preserve long-lived compatibility UI between the old shell and the new workspace.
- Cleanup/removal of obsolete surfaces must be an explicit backlog item, not an afterthought.

## Execute-Item Structure Chosen
1. Research contract to lock the new IA, graph model, lenses, inspector actions, and migration plan.
2. Graph data foundation to extract/adapt reusable graph primitives and formalize swarm-manager graph data/contracts.
3. Workspace shell replacement to introduce the sidebar/canvas/inspector frame.
4. Topology + scenario impact views.
5. Lifecycle/flow + operations views.
6. Explicit cleanup item to remove legacy tabbed/dashboard assumptions.

## Dependency Notes
- `execute/swarm-manager-graph-data-foundation` depends on the research contract.
- `execute/swarm-manager-graph-workspace-shell` depends on the research contract.
- The topology and flow/ops items depend on both the graph data foundation and the new shell.
- Legacy-surface removal depends on the new topology and flow/ops views.

## High-Risk Areas To Validate Early
- The prompt-manager graph implementation is not drop-in reusable; extract only the reusable seam.
- Graph chaos must be controlled through default filters, clustering, node caps, and lens-specific defaults.
- Scenario impact resolution from acceptance allow/deny needs to be formalized rather than left as scattered UI logic.
- Runs need first-class deep links into agent-manager so the workspace is operationally useful.
- The rewrite should improve code quality; do not accumulate dead routes, duplicate stores, or compatibility branches.

## Questions Deferred To Workshop
- Exact node and edge taxonomy for the first graph contract.
- Whether shared graph primitives should move into a package or remain scenario-local with a cleaner seam.
- Exact default landing route and mobile interaction model after the shell swap.
- Which live operations metrics are useful in v1 versus later.
