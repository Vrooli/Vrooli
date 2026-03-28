# Implementation Plan: Swarm Manager Graph Data Foundation

## 1. Purpose

Build the `/api/v1/graph?lens=topology|flow|operations` endpoint and `/ws/graph?lens=operations` WebSocket endpoint that serve lens-specific node/edge projections for the graph workspace. This is the data foundation that all subsequent graph workspace UI items depend on.

## 2. Required Reading

```bash
prompt-manager skill read api-steer seam-discovery-and-enforcement test
```

Also read the research conclusion for canonical node/edge model and lens definitions:
```bash
swarm-manager backlog file-get --kind research --name swarm-manager-graph-workspace-contract --path conclusion.md
```

Reference the ecosystem-manager WebSocket implementation for broadcast patterns:
```bash
# Key files for WebSocket broker design:
# scenarios/ecosystem-manager/api/pkg/websocket/manager.go — fan-out broker (Manager struct, broadcast channel, RWMutex client map)
# scenarios/ecosystem-manager/api/pkg/tasks/coordinator.go — Broadcaster interface, event dispatch via ApplyTransition
```

## 3. Problem Statement

The swarm-manager API currently serves entity data through flat CRUD endpoints and a single `/api/v1/overview` aggregation endpoint. The overview endpoint returns backlog items, initiatives with rollup, and a dependency graph — but lacks:

- Capture nodes and classified_as edges
- Execution records with executes/follow_up edges
- Agent run nodes with spawned_run edges
- Scenario nodes with targets edges (resolved from acceptance_allow globs)
- Lens-specific filtering (topology vs flow vs operations subsets)
- React Flow-compatible output format (typed nodes with positions, typed edges with source/target)
- Real-time push updates (WebSocket) for the Operations lens

The graph workspace UI cannot be built until this data layer exists.

## 4. Scope

### In scope
- New `internal/graph/` package with graph projection logic
- `GET /api/v1/graph?lens=topology|flow|operations` endpoint
- `/ws/graph` WebSocket endpoint for real-time push (Operations lens)
- Server-side node/edge assembly from existing data sources (backlog store, initiatives, captures, execution runs, agent-manager proxy)
- Lens-specific filtering and default status exclusions
- React Flow-compatible response format
- Server-side event dispatch infrastructure using callback interface injection (optional setter pattern)
- Tests for graph projection and WebSocket streaming

### Out of scope
- Client-side graph rendering (separate UI item)
- Node position computation (client-side Dagre responsibility)
- Inspector actions or detail endpoints
- Initiative clustering logic (client-side)
- Mobile optimizations
- Legacy route redirects
- SSE support (WebSocket only — reuses proven ecosystem-manager patterns)

## 5. Current Technical Context

### Key files
| File | Role |
|---|---|
| `api/main.go` | Server setup, route registration, adapter wiring |
| `api/internal/overview/service.go` | Overview endpoint — loads all items, initiatives, dep graph |
| `api/internal/depgraph/graph.go` | Pure graph library — edges, blocked/unblocked, topo sort |
| `api/internal/backlog/types.go` | BacklogItem struct with AcceptanceAllow/AcceptanceDeny globs |
| `api/internal/backlog/store.go` | File-backed backlog persistence, LoadAll(kinds) |
| `api/internal/initiatives/service.go` | Initiative loading with rollup computation |
| `api/internal/captures/handler.go` | Capture model with classification struct (Items array) |
| `api/internal/execution/model.go` | Record with 9 statuses, parent_execution_id, run_id |
| `api/internal/execution/service.go` | Execution lifecycle — List(filters), Get(id), state transitions |
| `api/internal/execution/store.go` | FileStore backed by `.vrooli/execution-runs.json` |
| `api/internal/agentmanager/` | Agent-manager proxy — GetRunState(runID), SpawnBacklog() |
| `api/internal/scenarios/handler.go` | LoadAll() returns []Scenario with Status field |
| `scenarios/ecosystem-manager/api/pkg/websocket/manager.go` | Reference: WebSocket broker with fan-out, Gorilla upgrader |
| `scenarios/ecosystem-manager/api/pkg/tasks/coordinator.go` | Reference: Broadcaster interface for event dispatch |

### Existing patterns
- Services receive dependencies via constructor injection
- Adapter interfaces bridge packages to avoid circular imports
- Handlers are thin transport-edge orchestrators
- All entities are file-backed (JSON on disk)
- No existing WebSocket infrastructure in swarm-manager — ecosystem-manager has a proven implementation
- Execution records stored in single `.vrooli/execution-runs.json` file
- Scenarios loaded via CLI provider (`vrooli scenario list`)
- Optional dependencies use setter methods (e.g., `SetSettingsReader` in agentmanager)

### Data sources for graph nodes
| Node Type | Source | Access Pattern |
|---|---|---|
| BacklogItem | `backlog.Store.LoadAll(kinds)` | Returns all items, filterable by kind |
| Initiative | `initiatives.Service.List()` | Returns `[]InitiativeWithRollup` |
| Capture | `captures` handler (filesystem: `captures/{id}/capture.json`) | Extract CaptureLister interface in graph package, implement via adapter |
| Scenario | `scenarios.Handler.LoadAll()` | Returns `[]Scenario` with Name and Status |
| ExecutionRecord | `execution.Service.List(ctx, filters)` | Returns `[]Record`, filterable by status/kind |
| Run | `agentmanager.Service.GetRunState(runID)` | Per-run lookup only, no listing — derive run_ids from ExecutionRecord.RunID |

## 6. Target End State

### Graph endpoint
`GET /api/v1/graph?lens=topology|flow|operations` returns:

```json
{
  "nodes": [
    {
      "id": "backlog-item/execute/my-task",
      "type": "BacklogItem",
      "data": { "kind": "execute", "name": "my-task", "title": "...", "status": "ready", "priority": 3 },
      "position": { "x": 0, "y": 0 }
    }
  ],
  "edges": [
    {
      "id": "depends_on:execute/a->execute/b",
      "source": "backlog-item/execute/a",
      "target": "backlog-item/execute/b",
      "type": "depends_on"
    }
  ],
  "meta": {
    "lens": "topology",
    "node_count": 42,
    "edge_count": 38,
    "generated_at": "2026-03-27T22:00:00Z"
  }
}
```

Positions are all `{x:0, y:0}` — client-side Dagre computes layout.

### WebSocket endpoint
`/ws/graph?lens=operations` — client connects via WebSocket. Messages from server:

```json
{"type": "full-sync", "data": {"nodes": [...], "edges": [...]}, "timestamp": 1711580400}
{"type": "node-update", "data": {"id": "scenario/my-app", "type": "Scenario", "data": {"status": "running"}}, "timestamp": 1711580401}
{"type": "edge-add", "data": {"id": "spawned_run:exec-123->run-456", "source": "...", "target": "...", "type": "spawned_run"}, "timestamp": 1711580402}
{"type": "heartbeat", "data": {}, "timestamp": 1711580430}
```

Message types: `full-sync`, `node-update`, `node-add`, `node-remove`, `edge-add`, `edge-remove`, `heartbeat`.

### WebSocket connection lifecycle
- Client connects, server sends immediate `full-sync` message with current operations graph state
- Subsequent messages are incremental deltas
- Server sends `heartbeat` every 30 seconds to keep connection alive and detect stale clients
- Client reconnects on disconnect (standard WebSocket reconnect logic)
- On write error, server closes the connection and removes the client from the registry

## 7. Implementation Strategy

### Phase 1: Graph data model and projection service
1. Create `internal/graph/types.go` — Node, Edge, GraphResponse, Meta structs (React Flow compatible)
2. Create `internal/graph/interfaces.go` — Define data-source interfaces:
   - `BacklogLister` — LoadAll(kinds) returning items
   - `InitiativeLister` — List() returning initiatives with rollup
   - `CaptureLister` — ListCaptures() returning captures with classifications (extracted from captures handler filesystem logic, exposed via adapter)
   - `ScenarioLister` — LoadAll() returning scenarios
   - `ExecutionLister` — List(ctx, filters) returning execution records
   - `RunStateGetter` — GetRunState(ctx, runID) returning run state
3. Create `internal/graph/projection.go` — ProjectionService with lens-specific builders
4. Implement `buildTopology()`:
   - BacklogItem nodes (exclude completed/archived by default)
   - Initiative nodes
   - Capture nodes (only classified captures)
   - Scenario nodes
   - Edges: depends_on (from BacklogItem.DependsOn), member_of (BacklogItem→Initiative via Initiative field), classified_as (Capture→BacklogItem via exact kind+title match against existing backlog items), targets (BacklogItem→Scenario resolved from AcceptanceAllow glob prefix matching at graph-build time)
5. Implement `buildFlow()`:
   - BacklogItem nodes (only active: queued, in_progress, needs_review, needs_fixup)
   - ExecutionRecord nodes
   - Edges: depends_on, executes (ExecutionRecord→BacklogItem), follow_up (ExecutionRecord→ExecutionRecord via parent_execution_id)
6. Implement `buildOperations()`:
   - Scenario nodes (all, with live status)
   - ExecutionRecord nodes (only active: pending, scheduled, starting, running, needs_review, validating, needs_fixup)
   - Run nodes (derived from active ExecutionRecords with non-empty RunID, fetched via RunStateGetter)
   - Edges: targets (active BacklogItem→Scenario), executes (active ExecutionRecord→BacklogItem), spawned_run (ExecutionRecord→Run)

### Phase 2: Graph HTTP handler and route registration
1. Create `internal/graph/handler.go` — HTTP handler for `GET /api/v1/graph`
2. Parse `lens` query param (default: topology), validate against allowed values
3. Wire into `main.go` route registration using adapter interfaces (same pattern as overview service)
4. Return JSON response with nodes, edges, and meta

### Phase 3: WebSocket infrastructure and operations stream
1. Create `internal/ws/broker.go` — WebSocket broker adapted from ecosystem-manager patterns:
   - Uses `gorilla/websocket.Upgrader` for connection upgrade
   - Client map (`map[*websocket.Conn]bool`) with `sync.RWMutex` for thread-safe access
   - Buffered broadcast channel for async event distribution
   - `startBroadcaster()` goroutine: reads from broadcast channel, calls `broadcastToAll()`
   - `broadcastToAll()`: iterates clients with RLock, calls `WriteJSON()`, closes on write error
   - `BroadcastUpdate(eventType string, data any)`: wraps data in `{type, data, timestamp}` envelope, non-blocking send to broadcast channel
   - `HandleWebSocket(w, r)`: upgrades connection, registers client, sends initial `full-sync` with current operations graph, defers cleanup on disconnect
   - Heartbeat goroutine: sends `{type: "heartbeat"}` every 30 seconds
2. Define `Broadcaster` interface in graph package for testability:
   ```go
   type Broadcaster interface {
       BroadcastUpdate(event string, payload any)
   }
   ```
3. Create `internal/graph/stream.go` — WebSocket handler for `/ws/graph`
   - On connect: build current operations graph, send as `full-sync` message
   - Subscribe to broker for subsequent incremental events
4. Create `internal/graph/dispatch.go` — EventDispatcher interface + adapter:
   ```go
   type EventDispatcher interface {
       DispatchNodeUpdate(nodeType, nodeID string, data any)
       DispatchEdgeChange(action string, edge Edge)
   }
   ```
5. Inject EventDispatcher into execution service via optional setter method (`SetEventDispatcher`):
   - On execution status transitions (in startLocked, Cancel, handleSpecSyncComplete, etc.)
   - Emit node-update for ExecutionRecord status changes
   - Emit edge-add for spawned_run when RunID is set
   - If dispatcher is nil, calls are no-ops (existing tests unaffected)
6. Inject EventDispatcher into scenarios handler via optional setter method:
   - On scenario start/stop/restart success responses
   - Emit node-update for Scenario status changes
7. Wire WebSocket broker into main.go, register `/ws/graph` route, connect dispatch adapters

### Phase 4: Tests
1. Unit tests for projection builders (each lens):
   - `TestProjectTopology` — correct node types and edge types, status exclusions
   - `TestProjectFlow` — execution chains, follow_up edges, active-only filtering
   - `TestProjectOperations` — scenario/run nodes, spawned_run edges
2. Unit tests for edge resolution:
   - `TestClassifiedAsEdges` — capture→backlog item matching via exact kind+title
   - `TestTargetsEdges` — acceptance_allow glob → scenario name prefix matching
   - `TestMemberOfEdges` — backlog item initiative field → initiative node
3. Integration test for graph endpoint:
   - `TestGraphHandler` — full HTTP request, response shape validation
   - `TestGraphHandlerInvalidLens` — 400 for bad lens value
4. WebSocket tests:
   - `TestWSBroker` — client register/unregister, fan-out, cleanup on disconnect
   - `TestWSStream` — connect via WebSocket, receive full-sync, receive incremental event

## 8. Contract Decisions

### Node ID format
`{node-type-kebab}/{kind}/{name}` for backlog items, `{node-type-kebab}/{name-or-id}` for others.
Examples: `backlog-item/execute/my-task`, `initiative/my-initiative`, `scenario/my-app`, `capture/cap-abc`, `execution-record/exec-123`, `run/run-456`

### Edge ID format
`{edge-type}:{source-short}->{target-short}` where source/target-short is the node ID without the type prefix.
Example: `depends_on:execute/a->execute/b`, `classified_as:cap-abc->execute/my-task`

### Lens parameter
Query param `lens` with values: `topology`, `flow`, `operations`. Default: `topology`. Invalid values return 400.

### Response envelope
Top-level keys: `nodes`, `edges`, `meta`. No pagination — graph is bounded by store caps (~200 items, ~100 captures, ~100 execution records).

### WebSocket message format
JSON messages with `{type, data, timestamp}` envelope (matching ecosystem-manager pattern). Message types: `full-sync`, `node-update`, `node-add`, `node-remove`, `edge-add`, `edge-remove`, `heartbeat`. Each message data is JSON.

### WebSocket transport choice
Using WebSocket (not SSE) because:
- Proven pattern in the codebase (ecosystem-manager's Gorilla-based broker)
- Supports bidirectional communication if future needs arise (e.g., client requesting specific node subscriptions)
- Familiar Gorilla upgrader + fan-out broadcast pattern
- Consistent with existing infrastructure patterns across scenarios

### Capture listing interface
The captures handler currently manages captures via filesystem directly in the handler. The graph projection needs a `CaptureLister` interface defined in the graph package. Implement by creating an adapter that reads the captures directory and loads `capture.json` files — same pattern the handler already uses internally, extracted to satisfy the interface. This follows the existing adapter pattern (like overview service consuming backlog data).

### classified_as edge matching
Match captures to backlog items using exact kind+title comparison. For each classification item in a capture, look for an existing backlog item with matching kind and title. Only emit a classified_as edge if a matching backlog item exists. This avoids creating edges to non-existent nodes and is deterministic — if the user accepted the classification, the backlog item exists with that title.

### Run node data derivation
No listing endpoint exists for runs in agent-manager. Run nodes are derived from ExecutionRecord.RunID fields. For each active execution with a non-empty RunID, call `agentmanager.Service.GetRunState(ctx, runID)` to get live status. Parallelize with bounded concurrency (e.g., `errgroup` with limit 5). If agent-manager is unavailable, omit Run nodes and note in meta.

### EventDispatcher injection pattern
Use optional setter method (`SetEventDispatcher(d EventDispatcher)`) on execution service and scenarios handler. If nil, dispatch calls are no-ops. This matches the existing `SetSettingsReader` pattern in agentmanager and avoids changing constructor signatures or breaking existing tests.

## 9. Testing Plan

| Test | Type | What it verifies |
|---|---|---|
| `TestProjectTopology` | Unit | Correct nodes/edges for topology lens, status exclusions applied |
| `TestProjectFlow` | Unit | ExecutionRecord→BacklogItem edges, follow_up chains, active-only |
| `TestProjectOperations` | Unit | Scenario/Run nodes, spawned_run edges, graceful agent-manager absence |
| `TestClassifiedAsEdges` | Unit | Capture→BacklogItem edges from classification.items exact kind+title matching |
| `TestTargetsEdges` | Unit | BacklogItem→Scenario edges from acceptance_allow glob prefix resolution |
| `TestMemberOfEdges` | Unit | BacklogItem→Initiative edges from initiative field |
| `TestGraphHandler` | Integration | Full HTTP request, lens param parsing, JSON response shape |
| `TestGraphHandlerInvalidLens` | Integration | 400 response for invalid lens |
| `TestWSBroker` | Unit | Client add/remove, fan-out, cleanup on disconnect |
| `TestWSStream` | Integration | Connect via WebSocket, receive full-sync, receive incremental event |

All tests use interface-based mocks (no external dependencies). WebSocket integration tests use `httptest.NewServer` with a test client that connects via Gorilla's `websocket.Dial`.

## 10. Rollout / Validation Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./... -timeout 300s` passes
- [ ] `GET /api/v1/graph?lens=topology` returns expected node/edge structure
- [ ] `GET /api/v1/graph?lens=flow` returns execution chain data
- [ ] `GET /api/v1/graph?lens=operations` returns live operational data
- [ ] `/ws/graph?lens=operations` opens WebSocket connection and receives full-sync
- [ ] WebSocket stream delivers node-update messages when execution status changes
- [ ] Operations lens gracefully degrades when agent-manager unavailable
- [ ] `gofumpt -l .` reports no formatting issues
- [ ] `golangci-lint run` passes

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Agent-manager unavailable during operations lens build | Run nodes missing | Return operations graph without run nodes; add `agent_manager_available: false` to meta |
| Acceptance_allow glob resolution is expensive | Slow topology response | Simple prefix matching against scenario names — O(items × scenarios), both bounded |
| WebSocket connections accumulate without cleanup | Memory leak | Broker tracks clients in map; heartbeat detects stale connections; unregister on write error; cleanup in defer block |
| No existing WebSocket infrastructure in swarm-manager | Implementation effort | Adapt proven ecosystem-manager patterns (Manager struct, fan-out broadcaster, Gorilla upgrader) |
| Large execution history | Slow flow lens | Default filter to active statuses only; completed/failed executions excluded |
| Capture listing not exposed as interface | New code needed | Extract filesystem listing from captures handler into a CaptureLister interface via adapter |
| Concurrent RunState lookups to agent-manager | Latency spike | Bounded errgroup (limit 5), timeout per call (2s), omit on failure |
| Gorilla WebSocket dependency | New dependency | Gorilla is already used by ecosystem-manager; well-maintained, widely adopted |

## 12. Non-goals / Prohibited Patterns

- Do NOT compute node positions server-side — client-side Dagre handles layout
- Do NOT add pagination — graph is bounded by store caps
- Do NOT build compatibility with the overview endpoint — this is a clean replacement for graph consumers
- Do NOT add SSE support — WebSocket only (consistent with ecosystem-manager patterns)
- Do NOT add authentication/authorization — matches existing API pattern (no auth)
- Do NOT modify existing endpoint signatures — additive only, use adapter interfaces
- Do NOT use polling for event sources — use callback injection into services via setter methods
- Do NOT change constructor signatures for EventDispatcher injection — use optional setter pattern

## 13. Definition of Done

- [ ] New `internal/graph/` package with types, interfaces, projection service, handler, stream handler, and dispatch interface
- [ ] New `internal/ws/` package with WebSocket broker (adapted from ecosystem-manager patterns)
- [ ] `GET /api/v1/graph?lens=topology|flow|operations` returns React Flow-compatible nodes/edges
- [ ] `/ws/graph?lens=operations` delivers real-time events via WebSocket with heartbeat
- [ ] All 6 canonical node types represented (Capture, BacklogItem, Initiative, Scenario, ExecutionRecord, Run)
- [ ] All 7 canonical edge types implemented (depends_on, member_of, classified_as, executes, targets, follow_up, spawned_run)
- [ ] Default status exclusions per lens applied
- [ ] Graceful degradation when agent-manager unavailable
- [ ] Tests pass with `go test ./... -timeout 300s`
- [ ] Code formatted with `gofumpt` and passes `golangci-lint run`
