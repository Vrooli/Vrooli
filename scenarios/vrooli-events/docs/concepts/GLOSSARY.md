# Glossary

## Core Concepts

**Event Envelope**
The standard container for all events. Contains: event ID (structured), event type, source scenario, target scenario (optional), correlation ID (optional), payload (proto-serialized), and metadata (JSON). Defined in `packages/proto/schemas/vrooli-events/v1/domain/`.

**Structured Event ID**
A hierarchical identifier for events: `{scenario}.{domain}.{action}.{version}`. Enables precise subscription matching via glob patterns. Example: `swarm-manager.backlog.item-completed.v1`.

**Glob Pattern**
Segment-aware pattern matching used for event subscriptions and policy rules. `*` matches exactly one segment, `**` matches one or more segments. Example: `swarm-manager.backlog.**` matches all backlog events from swarm-manager.

**Correlation ID**
An opaque string that groups related events across scenario boundaries. All events in a distributed operation share the same correlation ID, enabling trace visualization.

## Policy Concepts

**Policy Rule**
A configuration that governs inter-scenario communication. Three types: access control, rate limit, and circuit breaker. Stored in vrooli-events and distributed to all scenarios via SSE policy push.

**Access Control Rule**
A policy rule that allows or denies calls between scenario pairs, optionally filtered by endpoint pattern. Rules have priority — higher priority wins on conflict. Default behavior (when no rules match) is configurable: allow-all or deny-all.

**Rate Limit Rule**
A policy rule that caps the number of requests between a scenario pair within a sliding time window. Supports burst allowance for short spikes.

**Circuit Breaker**
A policy rule that automatically stops calls to a failing scenario. Three states:
- **Closed** (normal) — calls proceed, failures counted
- **Open** (blocking) — all calls denied immediately, cooling down
- **Half-Open** (probing) — one call allowed through to test recovery

**Policy Cache**
A local, in-memory copy of relevant policy rules maintained by each scenario. Loaded at startup, updated in real-time via SSE. Enables zero-latency policy enforcement without network calls to vrooli-events.

**Dual-End Enforcement**
Policy is enforced at both sender (before making the call) and receiver (before processing the request). Defense-in-depth pattern where neither side trusts the other's cache exclusively.

## Integration Concepts

**EmittingResolver**
A decorator around the standard discovery `Resolver` that adds automatic event emission and sender-side policy enforcement. Drop-in replacement — no changes to calling code.

**PolicyMiddleware**
Importable Go HTTP middleware that enforces receiver-side policy by checking the `X-Source-Scenario` header against the local policy cache.

**Fire-and-Forget**
Event emission pattern where events are buffered in a local channel and sent asynchronously. The caller never blocks waiting for vrooli-events to acknowledge. Events may be dropped if the buffer overflows or vrooli-events is unreachable.

## Subscription Concepts

**Persistent Subscription**
A named, durable subscription rule stored in vrooli-events. Unlike ephemeral SSE connections, persistent subscriptions survive restarts and deliver events via webhook or reconnectable SSE. Used by notification-hub for event-driven notifications.

**Webhook Delivery**
A subscription delivery method where matching events are POSTed to a URL. Includes HMAC-SHA256 signature for verification, retry with exponential backoff, and delivery health tracking.

**Subscription Health**
Tracking of delivery success/failure rates per subscription. Subscriptions auto-disable after consecutive failures (circuit-broken state) and can be re-enabled manually.

## Storage Concepts

**WAL Mode**
SQLite's Write-Ahead Logging mode. Enables concurrent readers with a single writer, providing good read throughput for event queries while events are being ingested.

**Retention Window**
Configurable maximum age for stored events (default: 30 days). Events older than this are deleted by the pruning background goroutine.

**Size Cap**
Configurable maximum storage size (default: 2GB). When exceeded, oldest events are pruned regardless of age.

**Dual-Trigger Pruning**
Background goroutine that runs on a configurable interval (default: 6 hours) and prunes events that exceed either the retention window OR the size cap.
