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

### Seam 5: Scenario-Side SDK ↔ vrooli-events API

- **Interface**: `EmittingResolver` wrapping a discovery `Resolver`; `PolicyMiddleware` wrapping `http.Handler`
- **Contract**: `EmittingResolver` adds zero latency (policy check in-memory, emit via background goroutine); `PolicyMiddleware` reads `X-Source-Scenario` header (see `internal/headers/`)
- **Test strategy**: Unit tests with mock events client (`internal/resolver/resolver_emit_test.go`, `internal/middleware/policy_test.go`); integration tests with real vrooli-events
- **Location**: `scenarios/vrooli-events/internal/resolver/`, `scenarios/vrooli-events/internal/middleware/`, `scenarios/vrooli-events/internal/emitter/`, `scenarios/vrooli-events/internal/fallback/`, `scenarios/vrooli-events/internal/headers/` (SDK layer — pending promotion to a shared module so external scenarios can import it)
- **Shared discovery primitive**: `packages/api-core/discovery/resolve.go` provides the underlying `Resolver` that `EmittingResolver` decorates

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

### UI HTTP Seam (`globalThis.fetch`)

**Seam quality: Good** (formalized Phase 3 iter1, 2026-04-20)

`ui/src/lib/api.ts` has always relied on `globalThis.fetch` as its one network
boundary, but there was no reusable way for tests to program responses or
inspect requests — every would-be behavioral test had to hand-roll
`vi.spyOn(globalThis, "fetch")` plumbing. That friction is why 270 lines of
`ui/src/lib/api.test.ts` covered only type shapes, not actual call behavior.

- **Seam**: the WHATWG `fetch` global; `api.ts` never captures a fetch reference at
  import time, so swapping the global *after* module load works cleanly.
- **Test helper**: `src/test-utils/mockFetch.ts` — `mockFetch()` replaces
  `globalThis.fetch` with a programmable `vi.fn()`, exposes `.respondTo(match, response)`,
  `.rejectWith(match, error)`, and `.calls[]` for assertions, and provides `.restore()` for
  teardown. Unmatched calls return a 500 with a diagnostic body so they fail loudly.
- **Mock**: matchers accept a URL substring or RegExp and an optional method filter;
  the first matching program wins (insertion order).
- **Reference tests**: `src/test-utils/mockFetch.test.ts` (9 cases covering the helper
  contract) and `src/lib/api.behavior.test.ts` (fetchHealth / fetchEvents through the
  seam).
- **Why it matters**: enables behavioral coverage of request construction
  (query-string encoding, method, headers, body shape) and of error paths (non-ok
  status, thrown network errors) without a running API, without vi.mock churn, and
  without changing a single line of `api.ts`.

### UI SSE Seam (`globalThis.EventSource`)

**Seam quality: Good** (formalized Phase 3 iter1, 2026-04-20)

`subscribeSSE` in `ui/src/lib/api.ts` constructs an `EventSource` per subscription.
Previously the test bootstrap (`src/test-utils/setup.ts`) replaced `EventSource` with
a constructor that produced an instance whose `addEventListener`/`close` were `vi.fn()` —
enough to prevent jsdom errors, not enough to drive messages. Behavioral tests for
SSE consumers (the subscribe loop, reconnection, named-event handling) were therefore
impossible to write cleanly.

- **Seam**: the DOM `EventSource` global; `api.ts` constructs via `new EventSource(url)`
  and attaches listeners immediately, so a swapped constructor can capture both URL and
  listeners.
- **Test helper**: `src/test-utils/mockEventSource.ts` — `mockEventSource()` replaces
  the constructor, records each instance (URL + listener map + close/addEventListener
  mocks), and exposes `.emitMessage(data)`, `.emitNamed(name, data)`, `.emitError()` so
  tests can drive the consumer deterministically. Static enum constants
  (`CONNECTING`/`OPEN`/`CLOSED`) are preserved on the constructor.
- **Reference tests**: `src/test-utils/mockEventSource.test.ts` (10 cases covering the
  helper contract) and `src/lib/api.behavior.test.ts` (subscribeSSE URL building,
  message parsing, error forwarding, malformed-JSON resilience).
- **Discovery from the seam**: writing the first behavioral test revealed that
  `subscribeSSE` wires the same handler to both `addEventListener("message", ...)` and
  `es.onmessage`, so unnamed SSE messages fire the consumer twice. That behavior is now
  locked in by test and documented in `docs/internal/PROBLEMS.md` for a dedicated fix
  phase.

### Clock Seam (`store.SQLiteConfig.Now`)

**Seam quality: Good** (new in Phase 20 iter2)

`SQLiteConfig.Now` is an injectable `func() time.Time` used by `pruneByTime()` to compute the
retention cutoff. Tests inject a fake clock instead of sleeping through `MaxAge`.

- **Default**: `time.Now` (applied in `NewSQLiteStore` if `Now == nil`)
- **Used by**: `SQLiteStore.pruneByTime`
- **Test usage**: `TestPruneByTime`, `TestPruneByTime_WithinRetention`, `TestPruneIdempotent`,
  `TestMetaConsistencyThroughPruneCycle` — all inject `fakeNow := func() time.Time { return time.Now().Add(10 * time.Second) }`
- **Why it matters**: Eliminated three `time.Sleep(2*time.Second)` calls; store package test time
  dropped from ~6.5s to ~0.5s. Prune behavior is now deterministic and fast, enabling finer-grained
  edge-case coverage without real-time penalties.

Note: SQLite still uses its own `strftime('now')` for the `created_at` column, so the seam shifts
*cutoff* time only, not inserted-row timestamps. This is intentional — the pruning decision is the
behavior we need to test, and tests don't need to freeze individual row timestamps to verify it.

### Weak/Missing Seams

| Location | Issue | Mitigation |
|----------|-------|------------|
| `api/main.go` main() | Directly constructs SQLiteStore, Broker, Pruner | Acceptable for entry point; all constructors accept interface deps |
| `api/handlers.go` protoMarshaler/protoUnmarshaler | Package-level globals | Low risk; stateless, thread-safe config objects |
| `ui/src/lib/api.ts` API_BASE | Module-level singleton resolved at import time | ✅ Mitigated (Phase 3, 2026-04-20): behavioral tests swap the real seam — `globalThis.fetch` — via the new `mockFetch` helper in `src/test-utils/`. Per-test base-URL override is still not supported, but no test has needed it; assertions against the observed request URL cover the same ground. |
| `store/sqlite.go` `strftime('now')` | Row-level timestamps still use SQLite's wall clock | Acceptable — cutoff seam is enough to make prune tests deterministic |

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
| UI api.ts fetchers | `mockFetch()` (swap `globalThis.fetch`) | Real UI smoke via browserless |
| UI subscribeSSE | `mockEventSource()` (swap `globalThis.EventSource`) | Real UI smoke via browserless |

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

## Architecture Alignment

Captures drift between the documented mental model and the actual physical
structure, and the alignment moves recorded in each phase.

### Logical vs Physical Structure

Two concerns currently share the flat `internal/` tree:

| Logical layer | Physical packages | Imported by `api/`? |
|---------------|-------------------|---------------------|
| **Server core** — runs inside the API binary | `store`, `broker`, `policy`, `subscription`, `convert`, `match`, `config`, `sqlutil` | Yes |
| **Scenario-side SDK** — meant for *other* scenarios to compose | `emitter`, `resolver`, `middleware`, `fallback`, `headers` | No |
| **Test fixtures** (server-side only) | `testutil` | Yes (tests only) |

### Drift Findings

1. **SDK layer is structurally orphaned.** `internal/emitter`, `internal/fallback`,
   `internal/headers`, `internal/resolver`, `internal/middleware` are imported by
   zero packages outside the SDK cluster itself — and Go's `internal/` rules
   prevent external scenarios from importing them at all. They are canonical
   reference implementations that cannot yet fulfill their declared role.
2. **Doc drift (fixed 2026-04-20):** `ARCHITECTURE.md` and `SEAMS.md`
   previously located the SDK at `packages/api-core/discovery/` and marked
   OT-P1-004 through OT-P1-014 as "Not implemented" despite their
   implementations landing earlier. Both now point to the real scenario-local
   paths and mark the P1 targets as Done.
3. **Server-core ↔ SDK shared seam:** `internal/match` is consumed by the
   broker (server) and by the resolver/middleware (SDK). It stays pure + stateless
   so this cross-layer dependency is safe.

### Alignment Moves Made

- Updated `ARCHITECTURE.md` with a "Package Layout — Server Core vs Scenario-Side
  SDK" section that names every `internal/*` package and its layer.
- Updated `ARCHITECTURE.md` P1 target table so the stated status matches the
  scoring engine (234/234 targets passing) and points at the real code
  locations.
- Rewrote Seam 5 in this file to reflect the SDK's real `internal/*` home
  and the `api-core/discovery.Resolver` primitive it decorates.

### Recommended Next-Iteration Refactor (High-Risk)

Promote the SDK cluster to a shared module so scenarios can actually import it:

```
packages/api-core/eventbus-sdk/
├── emitter/         (← scenarios/vrooli-events/internal/emitter)
├── resolver/        (← scenarios/vrooli-events/internal/resolver)
├── middleware/      (← scenarios/vrooli-events/internal/middleware)
├── fallback/        (← scenarios/vrooli-events/internal/fallback)
└── headers/         (← scenarios/vrooli-events/internal/headers)
```

Constraints for the move:
- `match` stays under the scenario (pure function, zero dependencies) and the
  SDK imports it from the shared location — or it also moves to api-core.
- `policy` stays server-only; the SDK currently imports it for evaluator
  logic — that evaluator subset must come along (split the package) or the
  SDK must depend on a shared evaluator contract.
- Test factories in `internal/testutil` remain server-only; the SDK needs
  its own test fixtures so consumers can plug in mock events clients without
  pulling in `MockStore`/`MockBroker`.

Estimated scope: 5 packages moved, ~60 import-path rewrites, re-run all
tests, add Go module boundary tests to prevent server packages from
leaking into the SDK. Deliberately deferred — documenting this here so the
next agent picks up with full context.

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
