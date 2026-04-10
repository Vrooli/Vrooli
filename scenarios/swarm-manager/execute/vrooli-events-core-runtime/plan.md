# Implementation Plan: Build The vrooli-events Scenario Core Runtime

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
swarm-manager backlog file-get --kind research --name vrooli-events-architecture --path conclusion.md
swarm-manager backlog file-get --kind execute --name vrooli-events-proto-schemas --path spec.json
```

## 1. Purpose

Build the greenfield `scenarios/vrooli-events/` Go scenario — the central event bus for all inter-scenario communication in Vrooli. This scenario provides:
- A durable SQLite event store with WAL mode for high-throughput append-only writes
- An SSE-based pub/sub server for real-time event delivery and policy push
- An async, non-blocking event ingestion HTTP endpoint
- Dual-trigger pruning (time + size) to keep storage bounded
- A CLI for event queries and subscription management

This is the foundational runtime that all other vrooli-events initiative items depend on.

## 2. Problem Statement

Vrooli currently has no centralized event bus. Inter-scenario communication is stateless and fire-and-forget with no traceability, no event history, and no pub/sub mechanism. The discovery package resolves scenario URLs but does not emit or store events. There is no infrastructure for event-driven subscriptions (e.g., notification-hub reacting to backlog item completions).

The research item `research/vrooli-events-architecture` has completed and produced a detailed architecture with settled decisions on storage, SSE protocol, event envelope format, and policy model. The proto schemas item `execute/vrooli-events-proto-schemas` has completed, providing generated Go types for the event envelope, policy rules, and SSE messages. This execute item implements the core runtime.

## 3. Scope

### In Scope
- Greenfield Go scenario at `scenarios/vrooli-events/`
- SQLite event store (schema from research Finding 14)
- SSE pub/sub server with heartbeat, Last-Event-ID resume, backpressure
- HTTP event ingestion endpoint (async, non-blocking)
- Dual-trigger pruning goroutine (30-day + 2GB, 6-hour cycle)
- Glob-pattern subscription matching on structured event IDs
- CLI for event queries and subscription management (built on cli-core ScenarioApp)
- Health endpoint
- service.json lifecycle configuration
- Comprehensive tests (unit + integration with real SQLite)

### Out of Scope
- Policy CRUD API and enforcement middleware (separate item: `execute/vrooli-events-policy-api-and-middleware`)
- Discovery package EmittingResolver decorator (separate item: `execute/discovery-event-emission-and-policy-cache`)
- Proto schema creation (completed: `execute/vrooli-events-proto-schemas`)
- Analytics UI (separate item: `execute/vrooli-events-analytics-ui`)
- UI components of any kind
- Authentication/authorization on endpoints (deferred)

## 4. Dependencies

| Dependency | Kind | Status | What It Provides |
|---|---|---|---|
| `research/vrooli-events-architecture` | research | **completed** | Architecture decisions, schema designs, protocol specs |
| `execute/vrooli-events-proto-schemas` | execute | **completed** | Proto-generated Go types at `github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain` — EventEnvelope, SubscriptionRequest, EventNotification, PolicySnapshot, HeartbeatMessage, AccessRule, RateLimit, CircuitBreaker |

Both dependencies are satisfied. No blockers.

## 5. Current Technical Context

### Proto Types Available
The generated Go package `domain` (import path: `github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain`) provides:
- `EventEnvelope` — event_id, source_scenario, target_scenario, event_type, timestamp, correlation_id, payload (anypb.Any), metadata (map[string]string)
- `SubscriptionRequest` — event_type_pattern, source_scenario_pattern, target_scenario_pattern
- `EventNotification` — stream_sequence (int64), envelope (EventEnvelope)
- `HeartbeatMessage` — timestamp, dropped_count
- `PolicySnapshot` — version, generated_at, access_rules, rate_limits, circuit_breakers
- Policy types: `AccessRule`, `RateLimit`, `CircuitBreaker`, `PolicyMatcher`
- Enums: `Effect` (ALLOW/DENY), `CircuitBreakerState` (CLOSED/OPEN/HALF_OPEN)

### Key Files / Components
- **Discovery package**: `packages/api-core/discovery/resolve.go` — stateless resolver, future EmittingResolver wraps this
- **Swarm-manager event log**: `scenarios/swarm-manager/internal/eventlog/` — reference for fire-and-forget SQLite pattern
- **Swarm-manager graph broker**: `scenarios/swarm-manager/internal/graph/broker.go` — reference for SSE heartbeat, broadcast buffer, stale client removal
- **LPBS API**: `scenarios/landing-page-business-suite/api/` — reference for handler factory pattern, service.json lifecycle
- **Proto module**: `packages/proto/go.mod` — module `github.com/vrooli/vrooli/packages/proto`, Go 1.24
- **cli-core**: `packages/api-core/cli-core/` — ScenarioApp scaffolding, API client utilities, consistent argument handling

### Codebase Patterns to Follow
- Internal packages with thin mains: `internal/store/`, `internal/broker/`, `internal/events/`
- Handler factory pattern with `*Server` receiver
- Real SQLite for integration tests (no mocks for DB)
- `gofumpt` for formatting, `golangci-lint` for linting
- Go 1.22+ net/http ServeMux for routing (stdlib, no external router dependency)
- cli-core ScenarioApp for CLI commands (consistent Vrooli ecosystem UX)

## 6. Target End State

A running `scenarios/vrooli-events/` scenario with:
1. **Event Store**: SQLite WAL-mode database accepting event writes and supporting queries by type, source, correlation_id, and time range
2. **SSE Server**: `/api/v1/events/subscribe` endpoint delivering real-time events with glob-pattern filtering, 30s heartbeat, Last-Event-ID resume, and backpressure handling
3. **Ingestion Endpoint**: `POST /api/v1/events` accepting event envelope payloads, returning 202 Accepted
4. **Pruning**: Background goroutine every 6 hours enforcing 30-day retention + 2GB size cap (tracked via metadata table)
5. **CLI**: `vrooli-events query`, `vrooli-events subscribe`, `vrooli-events stats` commands built on cli-core ScenarioApp
6. **Health**: `GET /health` endpoint for lifecycle health checks
7. All tests passing, code formatted and linted

## 7. Implementation Strategy

### All Decisions Settled

| Decision | Selection | Round |
|---|---|---|
| Proto dependency strategy | **Direct proto types** (proto schemas completed) | R1→R2 |
| Size-based pruning | **Metadata tracking table** (`store_meta` with `total_payload_bytes`) | R1 |
| Package structure | **Internal packages with thin mains** (`internal/store/`, `internal/broker/`) | R1 |
| Glob matching | **Custom segment-aware matcher** (split on `.`, `*` = one segment, `**` = multiple) | R1 |
| Ingestion model | **Sync store, async broadcast** (SQLite write before 202, SSE broadcast via broker channel) | R1 |
| Subscriber model | **Anonymous ephemeral connections** with Last-Event-ID resume | R1 |
| Serialization | **Proto binary for storage BLOB, JSON (via protojson) for SSE wire format** | R2 |
| Graceful shutdown | **Context cancellation cascade** (root ctx → server.Shutdown → broker.Close → DB close) | R2 |
| Event store types | **Internal `store.Event` struct**, convert to/from proto at handler boundary | R2 |
| HTTP router | **Go 1.22+ net/http ServeMux** — stdlib method routing + path params, zero dependencies | R2 |
| CLI framework | **cli-core ScenarioApp** — consistent Vrooli ecosystem UX, API client scaffolding | R2 |

### Phase 1: Scaffold & Event Store
1. Create `scenarios/vrooli-events/` directory structure:
   ```
   scenarios/vrooli-events/
   ├── .vrooli/service.json
   ├── Makefile
   ├── api/
   │   ├── main.go
   │   ├── routes.go
   │   └── handlers.go
   ├── cli/
   │   └── main.go
   └── internal/
       ├── store/
       │   ├── store.go        # Store interface + Event type
       │   ├── sqlite.go       # SQLite implementation
       │   ├── sqlite_test.go  # Integration tests
       │   ├── pruner.go       # Dual-trigger pruning
       │   └── pruner_test.go
       ├── broker/
       │   ├── broker.go       # SSE broker
       │   ├── broker_test.go
       │   ├── matcher.go      # Glob pattern matching
       │   └── matcher_test.go
       └── convert/
           └── convert.go      # store.Event ↔ proto EventEnvelope
   ```
2. Initialize Go module (`go mod init github.com/vrooli/vrooli/scenarios/vrooli-events`)
3. Create `.vrooli/service.json` with lifecycle configuration (no external resource deps — SQLite is embedded)
4. Implement SQLite event store in `internal/store/`:
   - `Event` struct: plain Go type with ID, EventID, SourceScenario, TargetScenario, EventType, CorrelationID, Payload ([]byte), Metadata (map[string]string), CreatedAt
   - `Store` interface: `Insert(ctx, Event) error`, `Query(ctx, QueryFilters) ([]Event, error)`, `GetSince(ctx, lastID int64, limit int) ([]Event, error)`, `Prune(ctx) (PruneResult, error)`, `Close() error`
   - SQLite implementation with WAL mode pragmas
   - `store_meta` table tracking `total_payload_bytes` (updated in same transaction as INSERT/DELETE)
   - Schema DDL from research Finding 14
   - Integration tests using real SQLite (`:memory:` for speed, file-backed for WAL-specific tests)
5. Implement dual-trigger pruning in `internal/store/pruner.go`:
   - Background goroutine with configurable interval (default 6h)
   - Time-based: DELETE WHERE created_at < 30 days
   - Size-based: DELETE oldest rows until total_payload_bytes < 2GB
   - No VACUUM in hot path
   - Accepts context for graceful shutdown

### Phase 2: SSE Pub/Sub Broker
1. Implement SSE broker in `internal/broker/`:
   - `Broker` struct managing subscriber map
   - `Subscribe(ctx, SubscriptionRequest) <-chan SSEMessage` — registers subscriber, returns event channel
   - `Publish(Event)` — non-blocking fan-out to matching subscribers
   - 64-capacity buffered channels per subscriber
   - Drop+notify backpressure: if channel full, drop event, increment dropped_count for next heartbeat
   - 30-second heartbeat goroutine per subscriber (`: heartbeat\n\n` comment line with dropped_count if > 0)
   - `retry: 5000` sent on connection establishment
   - Stale client removal on write errors
   - Context-based shutdown: cancel subscriber contexts, close channels
2. Implement glob-pattern matcher in `internal/broker/matcher.go`:
   - Custom segment-aware: split event type on `.`
   - `*` matches exactly one segment, `**` matches one or more segments
   - Table-driven unit tests
3. Implement Last-Event-ID resume: on subscribe, if Last-Event-ID header present, query store.GetSince() and replay before switching to live stream

### Phase 3: HTTP API & Ingestion
1. Implement `api/main.go`:
   - Parse config from env vars (PORT, DB_PATH, PRUNE_INTERVAL, etc.)
   - Initialize SQLite store, SSE broker, pruning goroutine
   - Root context with signal handling (SIGTERM, SIGINT)
   - Graceful shutdown cascade: cancel ctx → http.Server.Shutdown(30s) → broker.Close() → store.Close()
2. Implement routing using **Go 1.22+ net/http ServeMux**:
   - `mux.HandleFunc("POST /api/v1/events", s.handleIngest)` — ingest event
   - `mux.HandleFunc("GET /api/v1/events", s.handleQuery)` — query events
   - `mux.HandleFunc("GET /api/v1/events/subscribe", s.handleSubscribe)` — SSE subscription
   - `mux.HandleFunc("GET /health", s.handleHealth)` — health check
   - No external router dependency; stdlib ServeMux handles method routing and is sufficient for 4 endpoints
3. Implement handlers:
   - **Ingest**: Validate structured event ID format, parse JSON body into proto EventEnvelope (via protojson), convert to store.Event, insert synchronously, broadcast to broker asynchronously, return 202 Accepted
   - **Query**: Parse query params (type, source, correlation_id, since, limit), call store.Query(), convert results to proto, return JSON array via protojson
   - **Subscribe**: Parse query params into SubscriptionRequest, register with broker, set SSE headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`), stream events with `id:`, `event:`, `data:` fields
   - **Health**: Return 200 with basic status JSON
4. Error response format: `{"error": "message", "code": "ERROR_CODE"}`

### Phase 4: CLI
1. Build CLI on **cli-core ScenarioApp**:
   - Import `packages/api-core/cli-core` for ScenarioApp scaffolding
   - Use cli-core's API client utilities for HTTP calls to the vrooli-events API
   - Follow cli-core patterns for argument parsing, output formatting, and help text
2. Commands:
   - `query` — Search events by type/source/correlation_id/since/limit, output as table (default human-readable) or JSON
   - `subscribe` — Real-time SSE listener, print events as they arrive
   - `stats` — Show event store statistics (total events, DB size, last prune time)
3. CLI tests

### Phase 5: Integration & Polish
1. End-to-end test: ingest event → verify SSE delivery → verify store persistence → verify query retrieval
2. Create Makefile with standard targets (build, test, lint, fmt, start, stop, logs)
3. Verify `make start` / `make test` / `make stop` lifecycle
4. Format with `gofumpt`, lint with `golangci-lint`

## 8. Contract Decisions

### HTTP API
| Endpoint | Method | Description | Response |
|---|---|---|---|
| `/api/v1/events` | POST | Ingest event | 202 Accepted |
| `/api/v1/events` | GET | Query events with filters | 200 + JSON array |
| `/api/v1/events/subscribe` | GET | SSE subscription stream | 200 + text/event-stream |
| `/health` | GET | Health check | 200 + status JSON |

### Event Ingestion Request Body (JSON via protojson)
```json
{
  "eventId": "swarm-manager.backlog.item-completed.v1",
  "sourceScenario": "swarm-manager",
  "targetScenario": "notification-hub",
  "eventType": "swarm-manager.backlog.item-completed.v1",
  "correlationId": "abc-123",
  "payload": {"@type": "type.googleapis.com/...", "value": "..."},
  "metadata": {"key": "value"}
}
```
Note: protojson uses camelCase field names by default.

### SSE Stream Format
```
retry: 5000

id: 42
event: swarm-manager.backlog.item-completed.v1
data: {"eventId":"...","sourceScenario":"swarm-manager",...}

: heartbeat

: heartbeat dropped_count=3
```

### Query Parameters for GET /api/v1/events
- `type` — glob pattern filter on event_type
- `source` — exact match on source_scenario
- `correlation_id` — exact match
- `since` — event store autoincrement ID (for pagination/resume)
- `limit` — max results (default 100, max 1000)

### CLI Output
- Default: human-readable table output (per cli-steer guidance — agent prompts use CLI default human output, not --json)
- `--json` flag: JSON output for scripting/automation

## 9. SQLite Schema

```sql
CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT NOT NULL UNIQUE,
  source_scenario TEXT NOT NULL,
  target_scenario TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  correlation_id TEXT NOT NULL DEFAULT '',
  payload BLOB,
  metadata TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now'))
);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_source ON events(source_scenario);
CREATE INDEX idx_events_correlation ON events(correlation_id) WHERE correlation_id != '';
CREATE INDEX idx_events_created ON events(created_at);

CREATE TABLE store_meta (
  key TEXT PRIMARY KEY,
  value INTEGER NOT NULL DEFAULT 0
);
INSERT INTO store_meta (key, value) VALUES ('total_payload_bytes', 0);
```

## 10. Testing Plan

| Test Category | What | How |
|---|---|---|
| Event store unit | Insert, query, GetSince, prune operations | Real SQLite (`:memory:` for speed) |
| Store meta tracking | total_payload_bytes accuracy after insert/delete | Real SQLite with byte count assertions |
| SSE broker unit | Subscribe, broadcast, backpressure, heartbeat, stale removal | Go test with goroutines and channels |
| Glob matching unit | Pattern matching for subscriptions (* and ** segments) | Table-driven tests |
| Handler integration | Full HTTP request/response cycle | httptest.Server with real SQLite |
| Last-Event-ID resume | Reconnection replays missed events | SSE client disconnect/reconnect test |
| Pruning | Time-based and size-based retention | SQLite with synthetic old events and large payloads |
| E2E | Ingest → SSE delivery → query verification | Full server startup with test client |
| Convert | store.Event ↔ proto EventEnvelope round-trip | Table-driven conversion tests |

## 11. Rollout / Validation Checklist
- [ ] `go build ./...` succeeds for api and cli
- [ ] `go test ./... -timeout 300s` passes all tests
- [ ] `gofumpt -l .` reports no formatting issues
- [ ] `golangci-lint run` passes
- [ ] `make start` brings up the scenario via lifecycle
- [ ] `make test` runs the test suite
- [ ] Health endpoint responds 200
- [ ] Manual smoke: POST event → GET query returns it → SSE subscriber receives it

## 12. Risks + Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| SQLite single-writer bottleneck | Write throughput limit under high event volume | WAL mode + `busy_timeout=10000` handles moderate concurrency. Event ingestion is sync but fast (single-digit ms). Monitor and optimize if needed. |
| Size-based pruning accuracy | store_meta tracking can drift if process crashes mid-transaction | Use transactions for insert+meta-update atomically. On startup, run a one-time reconciliation query to correct drift. |
| SSE connection scaling | Many subscribers exhaust goroutines/memory | 64-capacity channels + drop policy bounds memory. Add subscriber limit if needed. |
| google.protobuf.Any serialization | First use of Any in codebase, potential edge cases | protojson handles Any marshaling with @type URLs. Test round-trip serialization thoroughly. |
| Go module dependency on packages/proto | Module path resolution across the monorepo | Use `replace` directive in go.mod to point to local proto package path. |

## 13. Non-goals / Prohibited Patterns
- Do NOT add policy CRUD or enforcement — that's a separate backlog item
- Do NOT modify the discovery package — that's a separate backlog item
- Do NOT use Postgres, Redis, or any external resource — SQLite only
- Do NOT add WebSocket support — SSE is the settled protocol
- Do NOT add authentication/authorization to endpoints in this phase
- Do NOT create a `lib/` directory — use v2.0 service.json lifecycle
- Do NOT mock SQLite in tests — use real SQLite instances
- Do NOT use gorilla/mux or chi — use Go 1.22+ net/http ServeMux

## 14. Definition of Done
- [ ] Greenfield scenario at `scenarios/vrooli-events/` with working service.json lifecycle
- [ ] SQLite event store with WAL mode, all CRUD operations, store_meta tracking, and dual-trigger pruning
- [ ] SSE pub/sub server with heartbeat, Last-Event-ID resume, glob filtering, and backpressure
- [ ] HTTP ingestion endpoint (POST /api/v1/events) returning 202 Accepted
- [ ] Query endpoint (GET /api/v1/events) with type/source/correlation_id/since/limit filters
- [ ] HTTP routing via Go 1.22+ net/http ServeMux (no external router)
- [ ] CLI with query, subscribe, and stats commands (built on cli-core ScenarioApp)
- [ ] Internal store.Event ↔ proto EventEnvelope conversion layer
- [ ] All tests passing with `go test ./... -timeout 300s`
- [ ] Code formatted with `gofumpt` and passing `golangci-lint`
- [ ] Scenario starts and stops cleanly via Makefile lifecycle
