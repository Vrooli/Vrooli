# Research Conclusion: Design The Event Bus Architecture And Policy Model

## Research Question
How should the vrooli-events architecture be designed to provide structured event emission, policy enforcement (access control, rate limiting, circuit breaking), SSE-based pub/sub, and durable event storage — all integrated with the existing discovery package — while adding zero latency to inter-scenario calls?

## Summary
The architecture builds on three proven codebase patterns: swarm-manager's fire-and-forget SQLite event log (append-only, WAL mode), swarm-manager's 3-state circuit breaker (threshold + cooldown + auto-reset), and the discovery package's stateless CLI-based resolution. The event bus uses a rich proto envelope with correlation_id and metadata map for traceability, typed proto messages for each policy kind (access control, rate limiting, circuit breaking), google.protobuf.Any for payload typing, SSE for server-to-client pub/sub with 30-second heartbeat, and a local ring buffer for event emission during outages. Event retention uses a dual-trigger pruning strategy (30-day window + 2GB size cap) with 6-hour background cleanup. Access control uses most-specific-wins evaluation with segment-count scoring. Rate limiting uses the token bucket algorithm. Policy cache invalidation uses full snapshot push via SSE. Event emission is injected via a decorator around the discovery Resolver, making it automatic for all scenarios.

## Methodology
- Audited the discovery package (`packages/api-core/discovery/resolve.go`) for resolution mechanics and extension points
- Surveyed all proto schemas under `packages/proto/schemas/` for existing event definitions and google.protobuf.Any usage
- Analyzed swarm-manager's `internal/eventlog/` as a reference implementation for fire-and-forget emission and SQLite storage
- Analyzed swarm-manager's `internal/execution/circuit_breaker.go` for circuit breaker state machine design
- Analyzed swarm-manager's `internal/graph/broker.go` for real-time messaging patterns (heartbeat, broadcast, stale client removal)
- Reviewed agent-manager's WebSocket protocol and event subscription model
- Searched for existing rate limiting, circuit breaking, and access control patterns across the codebase (none found)
- Reviewed the initiative orchestration summary for settled architectural decisions
- Audited discovery callsites across scenarios (app-monitor, swarm-manager) to identify the injection pattern for event emission
- Reviewed buf.yaml/buf.gen.yaml configuration for proto code generation pipeline

## Findings

### Finding 1: Discovery Package Is Stateless and Extension-Ready
The resolver (`packages/api-core/discovery/resolve.go`, 316 lines) shells out to `vrooli scenario port` on every call — no caching, no middleware hooks. The `Resolver` struct uses a `CommandRunner` interface, making it testable. Event emission should be injected as a **decorator pattern** (an `EmittingResolver` that wraps the existing Resolver), not at individual callsites. This keeps emission centralized and automatic — any scenario using the discovery package gets event emission for free.

**Hook points identified:**
- After successful resolution: emit `{scenario}.discovery.resolved.v1`
- On timeout: emit `{scenario}.discovery.timeout.v1`
- On scenario-not-running: emit `{scenario}.discovery.unavailable.v1`

**Callsite evidence:** Scenarios define wrapper functions (e.g., `resolveAgentManagerBaseURL()` in swarm-manager, `locateIssueTrackerAPIPort()` in app-monitor) that call `discovery.ResolveScenarioURL/Port`. The decorator sits below these wrappers, at the Resolver level.

### Finding 2: Swarm-Manager Event Log Is a Proven Pattern
Swarm-manager's `internal/eventlog/` implements exactly the fire-and-forget emitter pattern: `Emitter` struct with `Emit*` methods that log to SQLite with WAL mode, non-blocking, error-logged-but-not-propagated. The `Repository` interface abstracts storage. Schema uses auto-incrementing IDs, RFC3339Nano timestamps, entity-based organization, and JSON metadata as TEXT. This can be generalized into `packages/api-core/` as a shared event emission library.

**Current schema limitations for vrooli-events:**
- No `correlation_id` column (needed for cross-scenario tracing)
- No `source_scenario` / `target_scenario` columns (uses `entity_type` + `entity_id` instead)
- No retention/pruning logic — events accumulate indefinitely
- Single-connection pool (`SetMaxOpenConns(1)`) — sufficient for WAL mode but limits write throughput

### Finding 3: Proto Infrastructure Supports New Schemas
The `packages/proto/schemas/` directory uses `buf` for code generation into Go, TypeScript, Python, and JavaScript. Adding `vrooli-events/v1/` follows the established pattern (e.g., `agent-manager/v1/domain/events.proto`). Agent-manager's `RunEvent` proto with `oneof data` discriminated unions provides a template for the event envelope.

**google.protobuf.Any compatibility note:** Any is not currently used anywhere in the codebase (only Struct appears in ToolCallEventData.input and ErrorEventData.details). The decision to use Any for the event payload is correct — type URL-based runtime type checking is better than Struct for typed event payloads — but implementation will establish this pattern for the first time. The buf.yaml already depends on `google/protobuf` so no config changes are needed.

### Finding 4: Existing Circuit Breaker Provides Design Template
Swarm-manager's `internal/execution/circuit_breaker.go` implements a complete 3-state circuit breaker:
- **Closed** (normal): Entry absent or `BrokenAt` empty
- **Open** (tripped): `ConsecutiveFailures >= threshold`, `BrokenAt` set
- **Half-Open** (cooldown expired): Auto-reset when cooldown elapses
- Configurable via proto settings: `circuit_breaker_threshold` (default 3), `circuit_breaker_cooldown_minutes` (default 60)
- File-backed persistence with atomic JSON writes
- Manual reset endpoint available

For vrooli-events, this pattern adapts with:
- Track per-route failures (keyed by `{source_scenario}.{target_scenario}.{action}`) instead of per-item
- Add policy-driven thresholds (different routes can have different thresholds)
- Push state changes via SSE so both sender and receiver caches stay synchronized

### Finding 5: SSE Is Architecturally Simpler Than WebSocket for This Use Case
The initiative settled on SSE over WebSocket for pub/sub. Agent-manager uses WebSocket for bidirectional real-time streams (run events), but vrooli-events subscriptions are primarily server-to-client (event delivery + policy push). SSE is HTTP-native, auto-reconnects, and avoids the complexity of WebSocket connection management.

**Existing real-time patterns to adapt:**
- Swarm-manager's graph broker (`internal/graph/broker.go`) uses a 30-second heartbeat, 64-capacity broadcast buffer, non-blocking sends (drop if full), and stale client removal on write errors.

### Finding 6: SQLite + WAL Is the Storage Pattern
Swarm-manager uses SQLite with WAL mode for its event store with pragmas: `journal_mode(WAL)`, `busy_timeout(10000)`, `synchronous(NORMAL)`. The vrooli-events store uses the same configuration with added retention/pruning logic.

### Finding 7: Event Envelope Design (Settled)
The envelope uses the **rich** format: `event_id`, `source_scenario`, `target_scenario`, `event_type` (structured ID), `timestamp`, `correlation_id`, `payload` (google.protobuf.Any), `metadata` (map<string,string>). This provides cross-scenario traceability via correlation_id and schema-stable extensibility via the metadata map. OTel fields (span_id, parent_span_id) can live in metadata if needed later.

### Finding 8: Policy Rule Format (Settled)
Typed proto messages per policy kind: `AccessRule`, `RateLimit`, `CircuitBreaker`. Each has its own fields optimized for fast evaluation. Strong typing means no parsing at evaluation time — just field comparisons and enum checks. New policy kinds require proto schema changes, which is acceptable given the deliberate pace of policy evolution.

### Finding 9: Outage Behavior (Settled)
Local ring buffer with configurable max size (default: 1000 events or 10MB, whichever is hit first). During outage: events enqueue to the ring buffer. On reconnect: buffer flushes in order. If buffer fills: oldest events are dropped with a warning log. This balances data preservation during brief outages with bounded resource usage.

### Finding 10: No Retention or Pruning Exists Anywhere
The swarm-manager event log has no DELETE, VACUUM, or TTL logic — events accumulate indefinitely. The stats engine replays the entire event stream on startup via `Rebuild()`. For vrooli-events, pruning is essential since this handles cross-scenario traffic at much higher volume.

**Settled defaults:** 30-day time window, 2GB size cap, prune every 6 hours. Background goroutine runs: `DELETE FROM events WHERE created_at < datetime('now', '-30 days') OR (SELECT SUM(LENGTH(payload)) FROM events) > 2147483648` (oldest first when size cap exceeded).

### Finding 11: SSE Protocol Design (Settled)
- **Heartbeat**: `: heartbeat\n\n` comment line every 30 seconds (matches graph broker)
- **Reconnection**: `retry:` field set to 5000ms. Clients use `Last-Event-ID` header on reconnect to resume from the event store's autoincrement `id` column.
- **Backpressure**: Non-blocking send with bounded channel (64 capacity). If a subscriber falls behind, events are dropped for that subscriber. A `dropped_count` field in the next heartbeat notifies the subscriber.
- **Event format**: Standard SSE with `id:` (event store sequence number), `event:` (event type for client-side filtering), `data:` (JSON-encoded event envelope)

### Finding 12: Access Control Uses Most-Specific-Wins with Segment-Count Scoring (Settled)
When multiple access control rules match an event, the most specific rule wins. Specificity is determined by **segment-count scoring**: exact match = 3 points, prefix glob = 2 points, wildcard = 1 point, summed across source_pattern + target_pattern + action_pattern. Maximum score is 9 (all exact). Ties are broken by earliest creation time. This is intuitive for policy authors — set broad defaults with wildcards, add specific exceptions with exact patterns that automatically take precedence.

### Finding 13: Payload Uses google.protobuf.Any (Settled)
The event payload field uses `google.protobuf.Any`, which wraps a typed proto message with an embedded type URL. Consumers can type-check at runtime using the type URL. This is proto-native dynamic typing with safety guarantees. Code generation provides helper methods (Pack/Unpack). This will be the first use of Any in the Vrooli codebase.

### Finding 14: Proposed SQLite Event Store Schema
Based on all settled decisions, the event store DDL:

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
```

The `id` column doubles as the SSE `Last-Event-ID` for reconnect-and-resume. The `created_at` index supports time-based retention pruning. `payload` is BLOB (proto-serialized Any). `metadata` is TEXT (JSON-serialized map).

### Finding 15: Proposed Policy Rules Schema

```sql
CREATE TABLE policy_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL CHECK(kind IN ('access','rate_limit','circuit_breaker')),
  source_pattern TEXT NOT NULL DEFAULT '*',
  target_pattern TEXT NOT NULL DEFAULT '*',
  action_pattern TEXT NOT NULL DEFAULT '*',
  priority INTEGER NOT NULL DEFAULT 0,
  rule_data TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now'))
);
```

The `rule_data` column holds JSON-serialized typed rule payload (AccessRule, RateLimit, or CircuitBreaker). Pattern columns use glob syntax. Priority column stores the computed specificity score for access control rules.

### Finding 16: Discovery Decorator Pattern for Event Emission
Event emission wraps the Resolver at the library level, not at individual callsites. An `EmittingResolver` struct wraps the existing `Resolver` and emits events after each resolution attempt. This is transparent to scenarios — they continue calling `discovery.ResolveScenarioURL()` and events are emitted automatically.

### Finding 17: Rate Limiting Uses Token Bucket Algorithm (Settled)
No rate limiting exists in the codebase today. The chosen algorithm is **token bucket**: fixed capacity refilled at a steady rate, allowing bursts up to bucket capacity while enforcing an average rate. State per rule is minimal — just `tokens_remaining` and `last_refill_time` — making it efficient for dual-end cache storage. This is the industry standard for API rate limiting (used by AWS, Stripe) and avoids the fixed-window boundary spike problem. Each `RateLimit` policy rule specifies `capacity` (max burst size) and `refill_rate` (tokens per second).

### Finding 18: Policy Cache Invalidation Uses Full Snapshot Push (Settled)
When any policy rule changes on the vrooli-events server, the **entire policy ruleset** is pushed to all connected clients via SSE. Clients replace their local cache atomically — no partial state, no version tracking, no gap detection. This is bandwidth-acceptable at the current scale (dozens of rules, not thousands). If the ruleset grows large enough to make full pushes expensive, an upgrade path to incremental deltas exists but is not designed here.

## Limitations
- Circuit breaker thresholds for inter-scenario routes need real workload data to tune — starting with swarm-manager's defaults (3 failures, 60-min cooldown) is reasonable but may need per-route adjustment
- Event volume projections are estimated, not measured — retention defaults (30-day, 2GB) should be monitored and adjusted based on actual traffic
- SSE backpressure strategy (drop + notify) is simple but may lose important events for slow subscribers — an upgrade path to per-subscriber replay from the event store exists but is not designed here
- The local ring buffer implementation details (in-memory vs. on-disk, thread safety, flush ordering) need specification during implementation
- google.protobuf.Any is untested in this codebase — implementation may surface unexpected code generation or serialization issues
- The pruning SQL for size-based caps needs benchmarking — `SUM(LENGTH(payload))` over millions of rows may be slow and might need a separate tracking table
- Full snapshot push for policy cache invalidation is acceptable at current scale but will need migration to incremental deltas if the ruleset grows to hundreds or thousands of rules

## Actions

### Action 1: Create backlog item — Draft proto schemas for vrooli-events/v1/
- **Kind**: execute
- **Title**: Draft vrooli-events proto schemas (envelope, policy rules, SSE messages)
- **Description**: Create `packages/proto/schemas/vrooli-events/v1/` with: (1) event envelope proto using google.protobuf.Any payload, correlation_id, metadata map; (2) typed policy rule protos (AccessRule with segment-count specificity scoring fields, RateLimit with capacity and refill_rate for token bucket, CircuitBreaker with threshold and cooldown); (3) SSE subscription/policy-push message types including full-snapshot policy push format. Follow established buf patterns from agent-manager/v1/. Run buf generate to produce Go/TS/Python bindings.
- **Initiative**: vrooli-events
- **Priority**: 1 (blocks other execute items)

### Action 2: Create backlog item — Implement core event store and SSE pub/sub
- **Kind**: execute
- **Title**: Build vrooli-events core runtime (event store + SSE server)
- **Description**: Implement the SQLite event store with the schema from Finding 14, WAL mode pragmas, and dual-trigger pruning (30-day + 2GB, 6-hour cycle). Add SSE pub/sub server with 30-second heartbeat, Last-Event-ID resume, 64-capacity subscriber channels, and drop+notify backpressure. Include full-snapshot policy push on rule changes. This is the `execute/vrooli-events-core-runtime` item.
- **Initiative**: vrooli-events
- **Priority**: 1

### Action 3: Create backlog item — Implement discovery event emission decorator
- **Kind**: execute
- **Title**: Add EmittingResolver decorator to discovery package with policy cache
- **Description**: Create an EmittingResolver that wraps the existing Resolver in `packages/api-core/discovery/`. Emit structured events on resolve/timeout/unavailable. Add sender-side policy cache loaded at startup and updated via SSE full-snapshot push. Implement token bucket rate limiting and segment-count access control evaluation in the sender-side cache. Includes ring buffer for outage resilience. This is the `execute/discovery-event-emission-and-policy-cache` item.
- **Initiative**: vrooli-events
- **Priority**: 2 (depends on core runtime)

### Action 4: Create backlog item — Build policy management API and receiver middleware
- **Kind**: execute
- **Title**: Policy CRUD API and receiver-side enforcement middleware
- **Description**: REST API for managing policy rules (CRUD on policy_rules table from Finding 15). Receiver-side HTTP middleware using standard Go http.Handler chain pattern for access control (most-specific-wins with segment-count scoring), token bucket rate limiting, and circuit breaking enforcement. SSE full-snapshot policy push to connected clients on rule changes.
- **Initiative**: vrooli-events
- **Priority**: 2 (depends on core runtime)

<!-- Note: Actions 2 and 3 map to existing initiative items. Actions 1 and 4 are new items identified through this research. -->
