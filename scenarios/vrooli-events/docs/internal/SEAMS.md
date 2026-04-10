# Seams & Boundaries

Integration points, responsibility zones, and testability boundaries for vrooli-events.

## Scenario Boundary

vrooli-events owns:
- Event storage, querying, and retention
- SSE pub/sub for events and policy updates
- Policy rule storage and evaluation logic
- Persistent subscription management and webhook delivery
- Analytics aggregation and violation logging

vrooli-events does NOT own:
- Discovery package modifications (`packages/api-core/discovery/`) — that's a shared package change tracked by `execute/discovery-event-emission-and-policy-cache`
- Notification delivery — that's notification-hub's responsibility
- Event schema definitions per scenario — each scenario defines its own proto types
- Proto code generation — managed by `packages/proto/Makefile`

## Integration Seams

### Seam 1: Event Ingestion API ↔ Producers

- **Interface**: `POST /api/v1/events` with proto-JSON EventEnvelope
- **Contract**: `packages/proto/schemas/vrooli-events/v1/domain/events.proto`
- **Test strategy**: HTTP handler tests with mock store; integration tests with real SQLite
- **Caller**: EmittingResolver (fire-and-forget), any scenario via direct HTTP

### Seam 2: SSE Event Stream ↔ Consumers

- **Interface**: `GET /api/v1/events/subscribe` with SSE protocol
- **Contract**: SSE events with `id`, `event` type, `data` (JSON), heartbeat comments
- **Test strategy**: SSE client tests verifying event delivery, Last-Event-ID resume, backpressure behavior
- **Consumer**: notification-hub, UI dashboard, any scenario

### Seam 3: Policy SSE Push ↔ Scenario Caches

- **Interface**: `GET /api/v1/policies/subscribe` with SSE protocol
- **Contract**: `policy_snapshot` on connect, `policy_update` on changes, heartbeat with `policy_version`
- **Test strategy**: SSE client tests verifying snapshot delivery on connect, incremental updates on rule CRUD
- **Consumer**: EmittingResolver (sender cache), PolicyMiddleware (receiver cache)

### Seam 4: Webhook Delivery ↔ Subscription Targets

- **Interface**: HTTP POST to subscriber-configured URL
- **Contract**: JSON body = event, `X-VrooliEvents-Subscription` header, `X-VrooliEvents-Signature` HMAC
- **Test strategy**: Mock HTTP server receiving webhooks; verify signature, retry, and circuit-breaking behavior
- **Consumer**: notification-hub webhook endpoint

### Seam 5: Discovery Package ↔ vrooli-events API

- **Interface**: EmittingResolver wrapping Resolver; PolicyMiddleware wrapping http.Handler
- **Contract**: EmittingResolver adds zero latency; PolicyMiddleware reads X-Source-Scenario header
- **Test strategy**: Unit tests with mock events client; integration tests with real vrooli-events
- **Location**: `packages/api-core/discovery/`

## Code-Level Seams (Internal)

### Store Interface (`internal/store/store.go`)

**Seam quality: Excellent**

The `Store` interface (Insert, Query, GetSince, Prune, Stats, Close) is the primary testability boundary. All consumers (handlers, broker, pruner) depend on this interface rather than the concrete `SQLiteStore`.

- **Used by**: `api/handlers.go` (Server.store), `internal/broker/broker.go` (Broker.store), `internal/store/pruner.go` (PrunerConfig.Store)
- **Mock**: `internal/testutil/MockStore` — configurable via builder pattern (WithInsertResult, WithStatsError, etc.)
- **Real test impl**: In-memory SQLite via `NewSQLiteStore(ctx, SQLiteConfig{})` (empty DBPath = `:memory:`)

### EventBroker Interface (`internal/broker/iface.go`)

**Seam quality: Excellent** (new in Phase 3)

The `EventBroker` interface (Subscribe, Publish, SubscriberCount, DroppedCount, Close) decouples HTTP handlers from the concrete `*Broker` type. This allows handler unit tests to run without goroutines, channels, or real pub-sub infrastructure.

- **Used by**: `api/main.go` (Server.broker)
- **Mock**: `internal/testutil/MockBroker` — captures published events for assertion, configurable subscriber count
- **Why it matters**: Without this seam, every handler test required a real Broker (goroutine per subscriber, heartbeat tickers). Now handlers can be tested in isolation.

### Pattern Matching (`internal/match/match.go`)

**Seam quality: Perfect** — pure function, no state, no dependencies.

`match.Glob(pattern, value)` is used by both the SQLite store (post-SQL filter) and the broker (subscriber matching). Being a pure function, it's trivially testable with table-driven tests.

### Conversion Layer (`internal/convert/convert.go`)

**Seam quality: Good** — stateless data transformation between proto and store types.

Bidirectional conversion (EnvelopeToEvent / EventToEnvelope) separates the HTTP/proto layer from internal storage representation.

### Weak/Missing Seams

| Location | Issue | Mitigation |
|----------|-------|------------|
| `api/main.go` main() | Directly constructs SQLiteStore, Broker, Pruner | Acceptable for entry point; all constructors accept interface deps |
| `api/handlers.go` protoMarshaler/protoUnmarshaler | Package-level globals | Low risk; stateless, thread-safe config objects |
| `ui/src/lib/api.ts` API_BASE | Module-level singleton resolved at import time | Cannot override per-test; future improvement: accept base URL param |
| `ui/src/lib/router.ts` | Directly reads/writes window.location.hash | Must test with real DOM; future: abstract location object |

## Observability Surface

### Key Observable States

| State | Where Surfaced | Signal |
|-------|---------------|--------|
| API healthy | `GET /health` → 200 | `status: "healthy"`, `readiness: true` |
| Store down | `GET /health` → 503 | `status: "unhealthy"`, `readiness: false`, `error` field |
| SSE connected | UI status dot | Green pulse indicator + "Connected" |
| SSE disconnected | UI status dot | Red dot + "Disconnected — will auto-reconnect" |
| Backpressure | SSE heartbeat | `dropped_count=N` in heartbeat data |
| Replay errors | Server stderr | `replay: N replayed, M skipped` log line |
| Prune activity | Server stderr | Event count log on successful prune |
| API errors | HTTP response | JSON `{error, code}` with HTTP status |
| UI fetch errors | ErrorAlert component | Categorized message + retry button |
| Render crash | ErrorBoundary | Fallback UI with "Try Again" |

### Signal Inventory

**Server-side (stderr logs):**
- `insert error: <err>` — Store write failure
- `query error: <err>` — Store read failure
- `convert for broadcast: <err>` — Post-ingest broadcast failure
- `marshal for broadcast: <err>` — Post-ingest broadcast serialization failure
- `replay: invalid Last-Event-ID "<val>": <err>` — Bad reconnect header
- `replay: GetSince(lastID=N) failed: <err>` — Replay query failure
- `replay: convert event N: <err>` — Individual replay event failure
- `replay: N replayed, M skipped (from Last-Event-ID=X)` — Replay summary when events skipped

**Client-side (browser console):**
- `[SSE] Malformed event data: <preview>` — Parse failure on incoming SSE event
- `[SSE] Connection error` — EventSource error (auto-reconnect follows)

### Remaining Signal Debt

- No metrics/counters (e.g. events/sec, error rate) — acceptable for current scale
- Async broadcast errors (convert/marshal in goroutine) only logged, not surfaced to any status endpoint
- No structured log format (plain `log.Printf`) — sufficient for single-process scenario

## Testability Boundaries

| Component | Unit Test Approach | Integration Test Approach |
|-----------|--------------------|--------------------------|
| Event store | Mock SQLite (in-memory `:memory:`) | Real SQLite temp file |
| SSE broker | Mock subscriber channels | Real HTTP server + SSE client |
| HTTP handlers | MockStore + MockBroker (via EventBroker interface) | Real SQLite + Broker via httptest |
| Policy engine | Isolated evaluator with test rules | Full API → cache → evaluate cycle |
| Webhook delivery | Mock HTTP target server | Real target with test subscription |
| EmittingResolver | Mock events client, mock resolver | Real vrooli-events instance |
| PolicyMiddleware | Mock policy cache | Real SSE-fed cache |
| CLI | Mock API client | Real running scenario |

## Change Axes

Primary axes of evolution for vrooli-events, with current cost of change:

| Axis | Likely Change | Current Cost | Where Changes Land |
|------|---------------|-------------|-------------------|
| **New event fields** | Add fields to EventEnvelope (e.g. priority, ttl) | Low | `store/store.go` Event struct + `store/sqlite.go` schema/scan + `convert/convert.go` + proto schema |
| **New query filters** | Add filter params (e.g. target, date range) | Low | `store/sqlite.go` Query() + `api/handlers.go` parseQueryFilters() — centralized since Phase 7 |
| **New validation rules** | Require additional envelope fields | Very low | `api/handlers.go` validateEnvelope() — table-driven since Phase 7, single-line entry |
| **New error codes** | Add error categories for new failure modes | Very low | `api/handlers.go` error code constants — centralized since Phase 7 |
| **New subscriber patterns** | Filter on additional fields (e.g. correlation, metadata) | Medium | `broker/broker.go` subscriber struct + matches() + `broker/iface.go` SubscribeOpts |
| **New SSE event types** | Send policy updates, subscription events via SSE | Medium | New handler + route + broker channel type |
| **Retention policy changes** | Add per-type retention, rolling windows | Medium | `store/sqlite.go` Prune() + `config/config.go` |
| **Storage backend swap** | Replace SQLite with Postgres or other | Medium-High | Implement Store interface; all consumers already depend on interface |

### Stable Cores vs Volatile Edges

**Stable (change rarely):**
- `store.Store` interface — fundamental contract
- `broker.EventBroker` interface — pub-sub contract
- `match.Glob` — pure function, well-tested
- SSE protocol framing (`writeSSEMessage`) — SSE spec is fixed

**Volatile (change with new features):**
- `validateEnvelope` rules — grows with new required fields
- `parseQueryFilters` — grows with new filter params
- `config.Config` — grows with new tunable levers
- Error code constants — grows with new failure categories
- UI pages and components — grow with dashboard features

## Decision Points

Key decision points in the codebase and where they are made:

| Decision | Location | Inputs | Outcomes |
|----------|----------|--------|----------|
| **Envelope valid?** | `handlers.go:validateEnvelope()` | EventEnvelope fields | Accept (proceed to store) or reject (400 + error code) |
| **Query params valid?** | `handlers.go:parseQueryFilters()` | URL query params | Parsed filters or 400 error |
| **Glob vs exact query?** | `sqlite.go:Query()` | EventType contains `*` | Exact SQL match or LIKE + post-filter in Go |
| **Query limit clamping** | `sqlite.go:Query()` | Requested limit vs config | Clamped to [default, max] range |
| **Event matches subscriber?** | `broker.go:matches()` | Subscriber patterns + event fields | Deliver to channel or skip |
| **Channel full (backpressure)?** | `broker.go:Publish()` | Channel capacity | Deliver or drop + increment counter |
| **Heartbeat data content** | `broker.go:heartbeat()` | Dropped count | Empty heartbeat or heartbeat with `dropped_count=N` |
| **SSE frame type** | `handlers.go:writeSSEMessage()` | msg.Event field | SSE comment (heartbeat) or named event |
| **Store reachable?** | `handlers.go:handleHealth()` | Stats() error | 200 healthy or 503 unhealthy |
| **Replay on reconnect?** | `handlers.go:replayMissedEvents()` | Last-Event-ID header | Replay stored events or skip |
| **Payload unmarshal fallback** | `convert.go:EventToEnvelope()` | proto.Unmarshal result | Attach Any payload or silently drop |
| **Prune: time vs size** | `sqlite.go:Prune()` | Config thresholds + current state | Delete by age, then by size if over limit |

### Decision Groupings

- **Input validation**: `validateEnvelope`, `parseQueryFilters` — centralized in handlers.go
- **Routing/matching**: `matches()`, `Match()`, `Glob()` — layered pure functions
- **Resource protection**: channel backpressure, query limit clamping, body size limit — distributed but consistent pattern
- **Health/status**: `handleHealth`, heartbeat content — runtime observability decisions

## Responsibility Zones

```
┌─────────────────────────────────────────────────────────┐
│ vrooli-events scenario                                   │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ Event    │  │ Policy   │  │ Sub      │              │
│  │ Store    │  │ Engine   │  │ Manager  │              │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       │              │              │                    │
│  ┌────┴──────────────┴──────────────┴─────┐             │
│  │            HTTP API Layer              │             │
│  └────┬──────────────┬──────────────┬─────┘             │
│       │              │              │                    │
│  ┌────┴─────┐  ┌────┴─────┐  ┌────┴─────┐              │
│  │ SSE      │  │ Policy   │  │ Webhook  │              │
│  │ Broker   │  │ SSE Push │  │ Delivery │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
         ▲              ▲              │
         │              │              ▼
    Event SSE      Policy SSE     Webhook POST
    consumers      subscribers    to targets
```
