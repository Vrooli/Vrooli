# Implementation Plan: Flow and Operations Lens Views

## 1. Purpose

Implement the Flow lens and Operations lens in the swarm-manager graph workspace, adding lens-specific inspector actions and real-time update streaming for the Operations lens. Both lenses already have server-side projection support and the graph workspace shell is fully wired — this item focuses on the client-side lens behaviors, inspector actions, and live update plumbing.

## 2. Required Reading

```bash
prompt-manager skill read react-coherence api-steer implementation-plan-authoring
```

- `scenarios/swarm-manager/ui/src/surfaces/graph/` — all graph workspace components, stores, hooks, and lib
- `scenarios/swarm-manager/api/internal/graph/` — projection, streaming, broker, dispatch
- `scenarios/swarm-manager/api/internal/execution/` — handler.go (routes), review_client.go (git-control-tower integration)
- `scenarios/swarm-manager/api/main.go` — route registration and adapter wiring

## 3. Problem Statement

The graph workspace shell renders all three lenses using the same generic GraphNode and Inspector. Users cannot:
- See lifecycle progression (Flow lens) with appropriate actions (queue, retry, follow-up, trigger review, cancel)
- See live operational state (Operations lens) with control actions (start, stop, restart, view logs)
- Receive real-time graph updates when the Operations lens is active

The server-side projections and WebSocket infrastructure exist but the client does not consume them.

## 4. Scope

### In Scope
- Inspector action buttons per entity type per lens
- Client-side WebSocket integration for real-time graph updates on Operations lens
- Reconnection with exponential backoff + full-fetch fallback (full-replace strategy)
- Lens-specific default filters (if not already applied server-side)
- Visual differentiation between Flow and Operations lens nodes (status emphasis, activity indicators)
- Navigation actions that route to existing detail pages (BacklogDetailsPage, ScenarioDetailsPage)
- New detail pages for ExecutionRecord and PromptTrace if they don't already exist

### Out of Scope
- Topology lens changes (already working)
- Server-side projection changes (data-foundation handles this)
- New API endpoints beyond what exists (all required endpoints are confirmed present)
- Graph layout algorithm changes
- Settings/Prompts drawer changes

## 5. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `ui/src/surfaces/graph/components/Inspector.tsx` | Node detail panel — currently read-only with entity type badge, label, status, kind, and "View Details" link |
| `ui/src/surfaces/graph/components/GraphWorkspace.tsx` | Main orchestrator — fetches from 5 stores, assembles graph via `assembleGraphData()`, Dagre layout, 5s polling for runs |
| `ui/src/surfaces/graph/components/GraphNode.tsx` | Universal node renderer with entity-type color coding and status dot |
| `ui/src/surfaces/graph/stores/graph-data-store.ts` | Zustand store: nodes, edges, lens, entityFilters. Types: `Node[]`, `Edge[]`, `GraphLens`, `Record<EntityType, boolean>` |
| `ui/src/surfaces/graph/stores/graph-ui-store.ts` | Zustand store: selectedNodeId, highlightState, layoutMode, viewport, sidebarCollapsed, inspectorOpen |
| `api/internal/graph/stream.go` | WebSocket handler at `/ws/graph` — upgrades to WS, sends full-sync on connect |
| `api/internal/graph/broker.go` | Thread-safe client registry, 64-item broadcast buffer, 30s heartbeat, non-blocking send |
| `api/internal/graph/dispatch.go` | Event types: full-sync, node-update, node-add, node-remove, edge-add, edge-remove, heartbeat |
| `api/internal/graph/projection.go` | Builds 3 lens views from BacklogLister, ScenarioLister, ExecutionLister, RunStateGetter |
| `api/internal/execution/handler.go` | Execution routes including trigger-review |
| `api/internal/execution/review_client.go` | ReviewClient interface — calls git-control-tower's unified review API |

### Existing API Endpoints for Actions
| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/backlog/{kind}/{name}/queue` | Queue a backlog item for processing |
| `POST /api/v1/scenarios/{name}/start` | Start a scenario |
| `POST /api/v1/scenarios/{name}/stop` | Stop a scenario |
| `POST /api/v1/scenarios/{name}/restart` | Restart a scenario |
| `POST /api/v1/execution/{id}/cancel` | Cancel an execution |
| `POST /api/v1/execution/{id}/retry` | Retry an execution |
| `POST /api/v1/execution/{id}/follow-up` | Create a follow-up execution |
| `POST /api/v1/execution/{id}/trigger-review` | Trigger a GCT review for a terminal execution |

### Node ID Format
| Entity | ID Pattern | Example |
|--------|-----------|---------|
| Backlog | `backlog-item/{kind}/{name}` | `backlog-item/execute/my-feature` |
| Scenario | `scenario/{name}` | `scenario/swarm-manager` |
| Execution | `execution-record/{executionId}` | `execution-record/abc-123` |
| Agent Run | `agent-run/{runId}` | `agent-run/run-456` |
| Capture | `capture/{id}` | `capture/cap-789` |
| Initiative | `initiative/{name}` | `initiative/graph-workspace` |

### WebSocket Message Format
```typescript
interface WSMessage {
  type: "full-sync" | "node-update" | "node-add" | "node-remove" | "edge-add" | "edge-remove" | "heartbeat";
  data: any;
  timestamp: number;
}
```

### What's Already Working
- API: `/api/v1/graph?lens=topology|flow|operations` returns lens-filtered data
- API: `/ws/graph` WebSocket endpoint with broker + dispatch infrastructure
- API: Event dispatch wired into execution service and scenarios handler
- API: `POST /api/v1/execution/{id}/trigger-review` — git-control-tower review integration
- UI: GraphWorkspace fetches from API, renders via React Flow + Dagre
- UI: LensSwitcher with keyboard shortcuts (1-3)
- UI: Inspector opens on node click with entity type badge + "View Details"
- UI: BFS neighborhood highlight on selection
- UI: Sidebar with activity feed and entity filters

### What's Missing
- Inspector: No action buttons (queue, retry, start, stop, trigger review, etc.)
- WebSocket: Client-side does not connect or process WS messages
- No lens-specific visual treatment (e.g., running indicator on Operations nodes)
- No reconnection logic or full-fetch fallback
- Navigation actions for execution details and prompt traces (pages may need to be created)

## 6. Target End State

### Flow Lens
- Inspector shows contextual actions per entity type:
  - **BacklogItem**: Queue, View Execution History (navigate to BacklogDetailsPage), Follow-up
  - **ExecutionRecord**: View Prompt Trace (navigate to detail page), Follow-up, Retry, Trigger Review (calls git-control-tower via existing endpoint), Cancel
- Nodes emphasize status progression (queued → in_progress → needs_review → completed)
- Default filter: active statuses only (applied server-side, verified client-side)

### Operations Lens
- Inspector shows contextual actions per entity type:
  - **Scenario**: Start, Stop, Restart (POST endpoints), View Logs (navigate to ScenarioDetailsPage)
  - **ExecutionRecord**: View Prompt Trace (navigate to detail page), Cancel, View Logs (navigate)
  - **Run**: Stop (agent-manager API), View in Agent Manager (navigate)
- Real-time updates via WebSocket when lens is active
- WebSocket opened on lens switch to Operations, closed on switch away
- Automatic reconnection with exponential backoff (1s, 2s, 4s, 8s... max 30s)
- Full-replace fallback on reconnect (full-fetch from HTTP API replaces entire store)
- CSS-only activity pulse on nodes receiving WebSocket updates (2s fade ring animation)

### Navigation Actions
All "View X" actions navigate to existing detail pages where available:
- BacklogDetailsPage: `/backlog/{kind}/{name}`
- ScenarioDetailsPage: `/scenarios/{name}`
- ExecutionDetailsPage and PromptTracePage: create if they don't already exist (minimal pages that display the relevant data)

## 7. Implementation Strategy

### Phase 1: Inspector Actions Framework
1. Create action type definition in `ui/src/surfaces/graph/lib/`:
   ```typescript
   interface InspectorAction {
     id: string;
     label: string;
     icon: LucideIcon;
     variant: "default" | "destructive";
     handler: (node: Node) => Promise<void>;
     enabled?: (node: Node) => boolean;
   }
   type ActionRegistry = Record<GraphLens, Record<EntityType, InspectorAction[]>>;
   ```
2. Create `action-registry.ts` that defines all actions per lens × entity type
3. Node ID → API resource path parser utility (strip prefix, extract kind/name/id)
4. Wire action handlers to existing API endpoints via fetch calls
5. Update Inspector.tsx to render action buttons from registry based on active lens + selected node entity type
6. Inline error display: show error below action buttons, auto-dismiss after 5 seconds
7. Loading state per action button (disabled + spinner while in-flight)

### Phase 2: Flow Lens Actions + Navigation
1. Register Flow-specific actions:
   - BacklogItem: Queue (`POST /backlog/{kind}/{name}/queue`), View Execution History (navigate to BacklogDetailsPage), Follow-up
   - ExecutionRecord: Follow-up (`POST /execution/{id}/follow-up`), Retry (`POST /execution/{id}/retry`), Trigger Review (`POST /execution/{id}/trigger-review`), Cancel (`POST /execution/{id}/cancel`), View Prompt Trace (navigate)
2. Status progression visual: add a subtle left-border color gradient or step indicator on Flow lens nodes showing lifecycle stage
3. Verify server-side filters are respected (only active statuses visible)
4. Create ExecutionDetailsPage and PromptTracePage if they don't already exist (minimal detail pages)

### Phase 3: Operations Lens + Real-Time Updates
1. Create `useGraphWebSocket` hook in `ui/src/surfaces/graph/hooks/`:
   - Accept `enabled: boolean` parameter (true when Operations lens active)
   - Connect to `/ws/graph` when enabled
   - Parse incoming WSMessage and apply to graph-data-store:
     - `full-sync`: replace all nodes/edges via `setGraphData()`
     - `node-update`: update matching node data
     - `node-add`: add node to store
     - `node-remove`: remove node from store
     - `edge-add`/`edge-remove`: update edges
     - `heartbeat`: no-op (connection keepalive)
   - On disconnect: exponential backoff reconnect (1s base, 2x multiplier, 30s cap)
   - On reconnect: full-fetch from `/api/v1/graph?lens=operations`, full-replace store via `setGraphData()`, then resume WS
   - Cleanup: close WS on `enabled=false` or unmount
2. Register Operations-specific actions:
   - Scenario: Start, Stop, Restart (POST endpoints), View Logs (navigate to ScenarioDetailsPage)
   - ExecutionRecord: Cancel, View Prompt Trace (navigate), View Logs (navigate)
   - Run: Stop (agent-manager API), View in Agent Manager (navigate)
3. Activity pulse: CSS-only ring animation — add a `data-pulse` attribute / CSS class to the node wrapper when a WS `node-update` arrives, remove after 2s animation completes via `animationend` listener. Pure CSS `@keyframes`, no React re-render needed.

### Phase 4: Testing
1. **Action registry tests**: Verify correct actions returned for each lens × entity type combination
2. **Node ID parser tests**: Verify extraction of kind/name/id from all node ID formats
3. **WebSocket hook tests**: Mock WebSocket, verify connect/disconnect lifecycle, message processing, reconnection with backoff, full-replace on reconnect
4. **Inspector rendering tests**: Verify action buttons appear per lens and fire handlers on click
5. **Error handling tests**: Verify inline error display and auto-dismiss
6. **Navigation tests**: Verify navigation actions route to correct pages

## 8. Contract Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Real-time transport | WebSocket at `/ws/graph` (not SSE) | Already built by data-foundation. Zero server work needed. Spec originally said SSE but WS was built instead — use what exists. |
| Inspector action pattern | Centralized action registry `Record<Lens, Record<EntityType, Action[]>>` | Declarative, single source of truth, easy to test and extend |
| WebSocket lifecycle | Connect only when Operations lens active | Matches spec intent. No wasted bandwidth. Full-fetch on connect provides fresh state. |
| Action error UX | Inline error in Inspector, auto-dismiss 5s | Keeps error context close to the triggering action. No external toast dependency. |
| Navigation actions | Navigate to existing detail pages; create new pages (ExecutionDetailsPage, PromptTracePage) if needed | Consistent pattern: every "View X" navigates to a dedicated page. Keeps graph workspace focused on graph interaction. |
| Activity pulse | CSS-only ring pulse animation (2s fade) via transient CSS class | Lightest-weight approach — pure `@keyframes`, no React re-render of the node, zero performance cost. |
| Trigger Review | Use existing `POST /execution/{id}/trigger-review` endpoint | Endpoint already exists and calls git-control-tower's unified review API via `ReviewClient`. No new server work needed. |
| WS reconnection strategy | Full replace — call `setGraphData()` with fresh data | Simplest and most correct — guaranteed to match server state. Visual flash is acceptable for an exceptional event. |

## 9. Testing Plan

### Unit Tests
| Test File | Coverage |
|-----------|----------|
| `lib/action-registry.test.ts` | Action lookup per lens × entity type; enabled predicate logic; correct action counts per combination |
| `lib/node-id-parser.test.ts` | Parse all 6 node ID formats to extract API resource paths; edge cases (empty, malformed) |
| `hooks/useGraphWebSocket.test.ts` | Connect/disconnect lifecycle; message type handling (all 7 types); reconnection backoff timing; full-replace on reconnect; cleanup on unmount |

### Integration Tests
| Test File | Coverage |
|-----------|----------|
| `components/Inspector.test.tsx` | Renders correct action buttons per lens × entity type; click handlers dispatch API calls; inline error display and auto-dismiss; loading state during in-flight actions |

### Manual Verification
- Switch between all 3 lenses and verify Inspector shows correct actions
- Trigger each action and verify API call + UI feedback
- Trigger Review action dispatches to git-control-tower via existing endpoint
- Kill WS connection and verify reconnect with backoff + full-replace
- Verify no WS connections on Topology/Flow lenses
- Navigation actions route to correct detail pages
- Activity pulse appears on nodes receiving WS updates and fades after 2s

## 10. Rollout / Validation Checklist

- [ ] Flow lens shows correct actions per entity type in Inspector
- [ ] Operations lens shows correct actions per entity type in Inspector
- [ ] Actions dispatch to correct API endpoints and reflect state changes
- [ ] Trigger Review calls git-control-tower via existing endpoint
- [ ] WebSocket connects when Operations lens is active
- [ ] WebSocket disconnects when switching away from Operations
- [ ] Reconnection works with exponential backoff after connection drop
- [ ] Full-replace fallback fires on reconnect (store replaced, not merged)
- [ ] No WebSocket connections when on Topology or Flow lenses
- [ ] CSS activity pulse appears on nodes receiving WS updates, fades after 2s
- [ ] Inline error appears on action failure and auto-dismisses after 5s
- [ ] Navigation actions route to BacklogDetailsPage, ScenarioDetailsPage, ExecutionDetailsPage, PromptTracePage
- [ ] New detail pages created if they don't already exist
- [ ] Existing Topology lens behavior unchanged

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| WebSocket connection instability | Stale operations view | Exponential backoff (1s→30s cap) + full-replace from HTTP API on reconnect |
| Action handlers hitting unavailable endpoints (e.g., agent-manager down) | Failed actions with no feedback | Inline error display in Inspector; `enabled` predicate can disable actions when service unavailable |
| Node ID format mismatch between API and UI | Missing/broken actions | Node ID parser with exhaustive tests for all 6 formats; runtime type guard in action registry |
| Race between WS updates and user actions | Conflicting UI state | Full-replace on reconnect guarantees server-state convergence; individual WS updates are applied optimistically |
| Navigation to nonexistent detail pages | Broken navigation | Create ExecutionDetailsPage and PromptTracePage as part of Phase 2; verify target routes exist |
| git-control-tower unavailable when Trigger Review is invoked | Action fails silently | Inline error display surfaces the failure; ReviewClient already handles timeouts and errors gracefully |

## 12. Non-goals / Prohibited Patterns

- Do NOT modify server-side projection logic (data-foundation scope)
- Do NOT add new API endpoints — all required endpoints are confirmed present
- Do NOT create separate Inspector components per lens — use action registry pattern
- Do NOT poll for updates on Operations lens — use WebSocket only
- Do NOT keep WebSocket open when not on Operations lens
- Do NOT use merge strategy on WS reconnect — use full-replace via `setGraphData()`
- No compatibility shims or migration bridges
- No new Zustand stores — extend existing graph-data-store for WS state

## 13. Definition of Done

- Inspector renders lens-appropriate actions for all entity types in Flow and Operations lenses
- Actions successfully invoke their target API endpoints with correct resource paths
- Trigger Review action invokes git-control-tower via existing `POST /execution/{id}/trigger-review`
- Operations lens maintains a WebSocket connection with auto-reconnect and exponential backoff
- On reconnect, full-fetch from HTTP API replaces store (full-replace strategy)
- Graph updates in real-time when WS messages arrive on Operations lens
- CSS-only activity pulse animation (2s ring fade) on nodes receiving updates
- Inline error handling for failed actions with 5s auto-dismiss
- Navigation actions route to dedicated detail pages (creating ExecutionDetailsPage and PromptTracePage if needed)
- All new code has unit tests (action registry, node ID parser, WS hook, Inspector actions)
- No regressions in Topology lens behavior
