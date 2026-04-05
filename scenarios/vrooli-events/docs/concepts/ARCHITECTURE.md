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

[CODE: api/internal/store/events.go]

### SSE Broker

Real-time event delivery via Server-Sent Events. Features:

- **Glob-pattern filtering**: Subscribers specify patterns like `swarm-manager.backlog.**`
- **Last-Event-ID resume**: Clients reconnect without data loss
- **Backpressure**: 64-capacity channels with drop+notify to prevent slow subscribers from blocking
- **30s heartbeat**: Keeps connections alive and reports dropped event counts

[CODE: api/internal/broker/broker.go]

### Policy Engine

Three types of policy rules, all enforced locally from cache for zero-latency:

| Rule Type | Purpose | Enforcement |
|-----------|---------|-------------|
| Access Control | Allow/deny calls between scenario pairs | Immediate deny before network call |
| Rate Limiting | Cap requests per scenario pair per time window | Sliding window counter with burst allowance |
| Circuit Breaker | Auto-disable failing scenario pairs | Closed → Open → Half-Open → Closed lifecycle |

[REQ: REQ-POL-001, REQ-POL-002, REQ-POL-003]

### Dual-End Policy Caching

```
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

The `EmittingResolver` in `packages/api-core/discovery/` wraps the existing `Resolver`:

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

## Non-Goals

- **Clustering**: Single-node SQLite. Vrooli runs on a single server.
- **Exactly-once delivery**: Fire-and-forget with durable history for replay. Not Kafka.
- **Hard security boundary**: Policy is governance, not adversarial isolation. Scenarios in the Vrooli ecosystem trust each other.
- **External broker**: No RabbitMQ, Redis, NATS dependency. Self-contained.
