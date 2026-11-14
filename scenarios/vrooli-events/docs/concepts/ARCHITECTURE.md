# Architecture

## Overview

vrooli-events is the central nervous system for Vrooli's inter-scenario communication. It serves three complementary roles:

1. **Event Bus** — Durable storage and real-time delivery of structured events across scenarios
2. **Policy Engine** — Access control, rate limiting, and circuit breaking between scenarios with dual-end enforcement
3. **Analytics Platform** — Compliance auditing, throughput metrics, and distributed trace visualization

## System Components

### Event Store (SQLite WAL)

All events are durably stored in an embedded SQLite database using WAL mode for concurrent read access. The store supports:

- **Structured event IDs**: `{scenario}.{domain}.{action}.{version}`
- **Correlation tracking**: Events carry `correlation_id` for distributed trace grouping
- **Proto payloads**: Type-safe event data serialized via Protocol Buffers
- **Configurable retention**: Dual-trigger pruning by age (default 30 days) and size (default 2GB)

[CODE: internal/store/store.go]

### SSE Broker

Real-time event delivery via Server-Sent Events. Features:

- **Glob-pattern filtering**: Subscribers specify patterns like `swarm-manager.backlog.**`
- **Last-Event-ID resume**: Clients reconnect without data loss
- **Backpressure**: 64-capacity channels with drop+notify to prevent slow subscribers from blocking
- **30s heartbeat**: Keeps connections alive and reports dropped event counts

[CODE: internal/broker/broker.go]

### Policy Engine

Three types of policy rules, all enforced locally from cache for zero-latency:

| Rule Type | Purpose | Enforcement |
|-----------|---------|-------------|
| Access Control | Allow/deny calls between scenario pairs | Immediate deny before network call |
| Rate Limiting | Cap requests per scenario pair per time window | Sliding window counter with burst allowance |
| Circuit Breaker | Auto-disable failing scenario pairs | Closed → Open → Half-Open → Closed lifecycle |

[REQ: REQ-POL-001, REQ-POL-002, REQ-POL-003]

### Dual-End Policy Caching

```text
                    vrooli-events
                    (source of truth)
                         │
                    SSE policy push
                    ┌────┴────┐
                    ▼         ▼
              Sender cache   Receiver cache
              (EmittingResolver)  (PolicyMiddleware)
              │                   │
              ▼                   ▼
         Fail fast on deny   Reject unauthorized
         (save bandwidth)    (defense in depth)
```

**Why both sides?**
- **Sender cache**: Prevents wasted network calls. If policy says "don't call scenario B", the sender never makes the request.
- **Receiver cache**: Defense in depth. Even if a sender's cache is stale or a caller bypasses discovery, the receiver enforces its own policy.

Both caches load at startup and stay current via SSE. If vrooli-events goes down, last-known policy holds with configurable fail-open/fail-closed behavior.

[REQ: REQ-DI-003, REQ-DI-004, REQ-DI-005]

### Persistent Subscriptions

Named subscription rules that outlive SSE connections. Used by scenarios like notification-hub to durably subscribe to event patterns:

- **Glob patterns**: Same matching as SSE filters
- **Delivery types**: SSE (reconnectable) or webhook (HTTP POST with HMAC signature)
- **Health tracking**: Delivery success/failure rates, auto-disable on consecutive failures
- **Testing**: Synthetic test events to verify delivery before production use

[REQ: REQ-SUB-001 through REQ-SUB-005]

### Discovery Integration

The `EmittingResolver` in `scenarios/vrooli-events/internal/resolver/` wraps a discovery-style `Resolver`:

```
Scenario code calls: discovery.ResolveScenarioURL(ctx, "target-scenario")
                                    │
                              EmittingResolver
                              ├── 1. Check sender-side policy cache → deny fast if forbidden
                              ├── 2. Call underlying Resolver (existing behavior)
                              ├── 3. Fire-and-forget: emit discovery event (background goroutine)
                              └── 4. Return resolved URL to caller

Total added latency: ~0 (policy check is in-memory, event emission is async)
```

[REQ: REQ-DI-001, REQ-DI-002]

## Package Layout — Server Core vs Scenario-Side SDK

The code under `internal/` mixes two distinct concerns that benefit from being
read as separate layers. Keep this map in mind when deciding where new logic
belongs.

### Server Core — runs inside the `vrooli-events` API binary

Imported by `api/` (main binary) and by internal server code.

| Package | Role |
|---------|------|
| `internal/store/` | Durable event storage (SQLite WAL, `Store` interface) |
| `internal/broker/` | SSE pub/sub, matcher, `EventBroker` interface |
| `internal/policy/` | Policy rules storage, evaluator, SSE broadcaster |
| `internal/subscription/` | Persistent subscriptions + webhook delivery loop |
| `internal/convert/` | Proto ↔ store event conversion |
| `internal/match/` | Pure-function glob matcher (shared with sender-side SDK) |
| `internal/config/` | Server config resolution |
| `internal/sqlutil/` | Shared SQL helpers |

### Scenario-Side SDK — meant to be composed into **other** scenarios

Not imported by the server binary. Provides the caller-side and receiver-side
tooling every scenario needs to participate in the event bus + policy mesh.

| Package | Role |
|---------|------|
| `internal/emitter/` | Fire-and-forget event publisher with batching + retry |
| `internal/resolver/` | `EmittingResolver` — discovery decorator that emits + enforces sender-side policy |
| `internal/middleware/` | Receiver-side HTTP middleware: access, rate-limit, circuit-breaker |
| `internal/fallback/` | Fail-open / fail-closed behavior when the events API is unreachable |
| `internal/headers/` | `X-Source-Scenario` header injection + transport wrapper |

Because these packages currently live under `internal/`, Go blocks external
scenarios from importing them. The next-iteration refactor promotes the SDK
layer to a shared module (e.g. `packages/api-core/eventbus-sdk/`) so other
scenarios can consume the same implementation the tests already exercise.
Until that promotion lands, the SDK exists as canonical reference code and
is kept test-covered so the eventual move is mechanical.

### Shared Utilities

| Package | Role |
|---------|------|
| `internal/match/` | Pure glob matcher used by both server broker and SDK evaluator |
| `internal/testutil/` | `MockStore`, `MockBroker`, factories (server-side only — do not import from SDK packages) |

## Data Flow

### Event Ingestion

```
Producer → POST /api/v1/events → Validate → Insert SQLite → Notify SSE Broker
                                                                    │
                                                          ┌────────┴────────┐
                                                          ▼                 ▼
                                                    SSE subscribers   Webhook delivery
                                                    (real-time)       (persistent subs)
```

### Policy Update

```
Admin → POST/PUT /api/v1/policies → Store rule → Trigger SSE policy push
                                                          │
                                                    ┌─────┴─────┐
                                                    ▼           ▼
                                              All sender    All receiver
                                              caches        caches
                                              (immediate)   (immediate)
```

## Storage Schema

All data lives in a single SQLite database:

| Table | Purpose |
|-------|---------|
| `events` | Durable event store |
| `metadata` | Store-level stats (total_payload_bytes) |
| `settings` | Configurable retention, prune interval |
| `policy_rules` | Access control, rate limit, circuit breaker rules |
| `rate_limit_counters` | Sliding window counters for rate limiting |
| `circuit_breaker_state` | Current state per scenario pair |
| `policy_violations` | Audit log of denied requests |
| `subscriptions` | Persistent subscription definitions |
| `subscription_deliveries` | Webhook delivery log |

## Operational Target Traceability

Maps PRD operational targets to their implementation locations.

### P0 — Implemented

| Target | Description | Implementation |
|--------|-------------|----------------|
| OT-P0-001 | Event ingestion | [CODE: api/handlers.go#handleIngest], [CODE: internal/store/sqlite.go#Insert] |
| OT-P0-002 | SSE event subscribe | [CODE: api/handlers.go#handleSubscribe], [CODE: internal/broker/broker.go#Subscribe] |
| OT-P0-003 | Event query | [CODE: api/handlers.go#handleQuery], [CODE: internal/store/sqlite.go#Query] |
| OT-P0-004 | Health endpoint | [CODE: api/handlers.go#handleHealth] |
| OT-P0-005 | Structured event IDs | [CODE: internal/match/match.go#Glob], [CODE: internal/broker/matcher.go#Match] |
| OT-P0-006 | Correlation tracking | [CODE: internal/store/store.go#Event] (CorrelationID field) |

### P1 — Implemented

| Target | Description | Implementation | Status |
|--------|-------------|----------------|--------|
| OT-P1-001 | Last-Event-ID resume | [CODE: api/handlers.go#handleSubscribe] — reads `Last-Event-ID` header, replays via `store.GetSince` | Done |
| OT-P1-002 | Backpressure handling | [CODE: internal/broker/broker.go#Publish] — 64-cap channel, `droppedCount` in heartbeat | Done |
| OT-P1-003 | Configurable retention | [CODE: internal/store/pruner.go#StartPruner] — reads settings table | Done |
| OT-P1-004 | Policy — access control | [CODE: internal/policy/evaluator.go], [CODE: internal/middleware/policy_access.go] | Done |
| OT-P1-005 | Policy — rate limiting | [CODE: internal/middleware/policy_ratelimit.go], sliding window in [CODE: internal/policy/sqlite.go] | Done |
| OT-P1-006 | Policy — circuit breaking | [CODE: internal/middleware/policy_circuit.go], state in [CODE: internal/policy/sqlite.go] | Done |
| OT-P1-007 | Policy CRUD API | [CODE: api/handlers_policy.go] — `POST/GET/PUT/DELETE /api/v1/policies` + `/evaluate`, `/violations`, `/{id}/override` | Done |
| OT-P1-008 | SSE policy push | [CODE: api/handlers_policy.go#handlePolicySubscribe] — `GET /api/v1/policies/subscribe`; fan-out via [CODE: internal/policy/broadcaster.go] | Done |
| OT-P1-009 | Dual-end policy caching | Sender-side: [CODE: internal/resolver/resolver.go] (EmittingResolver). Receiver-side: [CODE: internal/middleware/policy.go] | Done |
| OT-P1-010 | Discovery integration | [CODE: internal/resolver/resolver.go] — EmittingResolver decorator, fire-and-forget emit via [CODE: internal/emitter/emitter.go] | Done |
| OT-P1-011 | Receiver-side middleware | [CODE: internal/middleware/policy.go] — composable HTTP middleware | Done |
| OT-P1-012 | Persistent subscriptions | [CODE: api/handlers_subscription.go], [CODE: internal/subscription/sqlite.go], webhook: [CODE: internal/subscription/webhook.go] | Done |
| OT-P1-013 | CLI tools | [CODE: cli/domains/events/register.go] — event commands via ScenarioApp | Done |
| OT-P1-014 | Graceful degradation | [CODE: internal/fallback/fallback.go] — fail-open / fail-closed modes on events API unavailability | Done |

### P2 — Planned

| Target | Description | Implementation | Status |
|--------|-------------|----------------|--------|
| OT-P2-001 | Analytics dashboard | [CODE: ui/src/pages/AnalyticsPage.tsx] — basic stats from health endpoint | Partial |
| OT-P2-002 | Policy management UI | Not implemented | Planned |
| OT-P2-003 | Subscription management UI | Not implemented | Planned |
| OT-P2-004 | Compliance & audit UI | Not implemented | Planned |
| OT-P2-005 | Event detail & trace view | [CODE: ui/src/components/EventDetail.tsx] — JSON viewer, no trace visualization | Partial |
| OT-P2-006 | Retention settings UI | [CODE: ui/src/pages/SettingsPage.tsx] — read-only display | Partial |
| OT-P2-007 | Circuit breaker dashboard | Not implemented | Planned |
| OT-P2-008 | Event replay | Not implemented | Planned |
| OT-P2-009 | Metrics export | Not implemented | Planned |

## Non-Goals

- **Clustering**: Single-node SQLite. Vrooli runs on a single server.
- **Exactly-once delivery**: Fire-and-forget with durable history for replay. Not Kafka.
- **Hard security boundary**: Policy is governance, not adversarial isolation. Scenarios in the Vrooli ecosystem trust each other.
- **External broker**: No RabbitMQ, Redis, NATS dependency. Self-contained.
