# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## Overview
- **Purpose**: Central nervous system for all inter-scenario communication in Vrooli. Every call through the discovery package emits a structured event to vrooli-events for durable storage, real-time pub/sub, analytics, compliance auditing, and policy enforcement. Scenarios subscribe to events without point-to-point coupling. Downstream consumers like notification-hub react to any event in the ecosystem without requiring changes to source scenarios.
- **Primary users/verticals**: Other Vrooli scenarios (as event producers/consumers), notification-hub (event-driven notifications), platform operators (analytics, compliance, policy management), developers (debugging event flows, tracing inter-scenario calls)
- **Deployment surfaces**: API (event ingestion + query + SSE streaming + policy CRUD + subscription management), CLI (query/subscribe/stats/policy/subscriptions), UI (real-time analytics dashboard, policy management, subscription management, compliance audit views)
- **Value promise**: Enables enterprise-grade event-driven architecture across scenarios with full traceability, durable history, real-time delivery, policy enforcement (access control, rate limiting, circuit breaking), and compliance auditing. The discovery package automatically emits events for every inter-scenario call — zero code changes needed in existing scenarios. Dual-end policy caching ensures zero-latency enforcement at both sender and receiver without adding round-trip overhead to inter-scenario calls.

## Operational Targets

### P0 -- Must ship for viability
- [x] OT-P0-001 | Event ingestion | POST /api/v1/events accepts EventEnvelope proto payloads, returns 202 Accepted, stores durably in SQLite WAL-mode
- [x] OT-P0-002 | SSE event subscribe | GET /api/v1/events/subscribe delivers real-time events with glob-pattern filtering on event type, source, and target
- [x] OT-P0-003 | Event query | GET /api/v1/events returns stored events with filters (type glob, source, correlation_id, since, until, limit, offset)
- [x] OT-P0-004 | Health endpoint | GET /health reports store stats, subscriber count, policy cache status, and overall health
- [x] OT-P0-005 | Structured event IDs | Events use `{scenario}.{domain}.{action}.{version}` format with segment-aware glob matching (* = one segment, ** = one or more)
- [x] OT-P0-006 | Correlation tracking | Events carry correlation_id for distributed trace grouping across scenario boundaries

### P1 -- Should have post-launch
- [x] OT-P1-001 | Last-Event-ID resume | SSE clients reconnect with Last-Event-ID header to replay missed events without data loss
- [x] OT-P1-002 | Backpressure handling | 64-capacity subscriber channels with drop+notify (dropped_count in heartbeat)
- [x] OT-P1-003 | Configurable retention | Dual-trigger pruning with user-configurable retention window (default 30 days) and size cap (default 2GB), manageable via API and UI
- [ ] OT-P1-004 | Policy engine — access control | Rules defining which scenarios can call which endpoints on which target scenarios, evaluated locally from cache
- [ ] OT-P1-005 | Policy engine — rate limiting | Per-scenario-pair rate limit rules (requests/minute, requests/hour) with sliding window counters
- [ ] OT-P1-006 | Policy engine — circuit breaking | Automatic circuit breaker per scenario pair: open after N failures in window, half-open probe after cooldown, close on success
- [ ] OT-P1-007 | Policy CRUD API | Full CRUD for policy rules (access control, rate limits, circuit breakers) via REST API with validation
- [ ] OT-P1-008 | SSE policy push | Dedicated SSE channel pushes policy updates to all connected scenarios in real-time so local caches stay current
- [ ] OT-P1-009 | Dual-end policy caching | Sender-side cache in discovery package (fail-fast on denied calls) + receiver-side middleware (defense-in-depth), both updated via SSE
- [ ] OT-P1-010 | Discovery package integration | EmittingResolver decorator wraps existing Resolver, auto-emits events for every inter-scenario call via fire-and-forget background goroutine with zero latency impact
- [ ] OT-P1-011 | Receiver-side policy middleware | Importable Go middleware that scenarios plug into their HTTP routers to enforce receiver-side policy from local cache
- [ ] OT-P1-012 | Persistent subscriptions | CRUD for named subscription rules with glob patterns, delivery target (SSE reconnect key or webhook URL), and enable/disable toggle
- [ ] OT-P1-013 | CLI tools | Full CLI via ScenarioApp: query, subscribe, stats, policy CRUD, subscription management, retention config
- [ ] OT-P1-014 | Graceful degradation | If vrooli-events is unreachable, discovery package holds last-known policy (configurable fail-open or fail-closed per rule), buffers events for retry

### P2 -- Future / expansion
- [ ] OT-P2-001 | Analytics dashboard | Real-time UI: event throughput charts, per-scenario call volume and error rates, top event types, subscriber status, store growth
- [ ] OT-P2-002 | Policy management UI | Visual policy editor: access control rules, rate limit configs, circuit breaker thresholds with status indicators (open/closed/half-open)
- [ ] OT-P2-003 | Subscription management UI | Create/edit/delete persistent subscriptions, test glob patterns against live event stream, subscription health indicators
- [ ] OT-P2-004 | Compliance & audit UI | Searchable policy violation log, per-scenario compliance scorecard, audit trail export, blocked request history
- [ ] OT-P2-005 | Event detail & trace view | Drill into individual events: full payload, metadata, correlation chain visualization across scenario boundaries
- [ ] OT-P2-006 | Retention settings UI | Configure retention window, size cap, and pruning schedule from dashboard with storage usage visualization
- [ ] OT-P2-007 | Circuit breaker dashboard | Live circuit breaker status per scenario pair, failure rate graphs, manual override (force open/close), cooldown configuration
- [ ] OT-P2-008 | Event replay | Re-emit historical events to a subscription for backfill or debugging scenarios
- [ ] OT-P2-009 | Metrics export | Prometheus-compatible /metrics endpoint for integration with external monitoring (Grafana, etc.)

## Tech Direction Snapshot
- Preferred stacks / frameworks: Go (API + CLI), React + Vite + Tailwind (UI), SQLite with WAL mode (storage)
- Data + storage expectations: Embedded SQLite (no external database dependency). WAL mode for concurrent reads with single writer. Metadata table tracks total_payload_bytes for size-based pruning. Policy rules stored in dedicated SQLite tables alongside events. Persistent subscriptions stored in SQLite. All configuration stored in SQLite settings table — no external config files needed beyond environment variables for bootstrap.
- Integration strategy: HTTP API for event ingestion, SSE for real-time event delivery and policy push. Discovery package integration via EmittingResolver decorator pattern — wraps existing Resolver with zero-latency event emission (fire-and-forget background goroutine) and local policy cache. Receiver-side middleware available as importable Go package. Proto-serialized payloads for type safety across all scenario boundaries.
- Non-goals / guardrails: No clustering or multi-node (single-node SQLite is sufficient for current scale). No guaranteed exactly-once delivery (fire-and-forget with durable history for replay). No message queuing or consumer group semantics (SSE pub/sub, not Kafka). No external message broker dependency. Policy enforcement is best-effort with local caches — not a hard security boundary (scenarios trust each other within the Vrooli ecosystem, policy is for governance and analytics, not adversarial isolation).

## Dependencies & Launch Plan
- Required resources: None (embedded SQLite, no external dependencies)
- Scenario dependencies: packages/proto (generated Go types for EventEnvelope, PolicyRule, SubscriptionRule, etc.), packages/api-core/discovery (integration target for EmittingResolver and policy middleware)
- Operational risks: SQLite single-writer bottleneck at very high throughput (mitigated by WAL mode and async ingestion). Policy cache staleness window between SSE disconnect and reconnect (mitigated by configurable fail-open/fail-closed). Fire-and-forget emission means events can be lost if vrooli-events is down and local buffer overflows (mitigated by bounded buffer with overflow logging).
- Launch sequencing:
  1. Core runtime — event store, SSE pub/sub, ingestion/query API, CLI (done)
  2. Policy engine — access control, rate limiting, circuit breaking, CRUD API, SSE policy push
  3. Discovery integration — EmittingResolver, sender-side policy cache, receiver-side middleware
  4. Persistent subscriptions — subscription CRUD, delivery management
  5. Analytics & compliance UI — dashboard, policy management, subscription management, audit views

## UX & Branding
- Look & feel: Dark theme primary, consistent with Vrooli platform aesthetics. Real-time charts with smooth animations for event throughput. Color-coded severity for policy violations (red), warnings (amber), and normal events (green). Monospace font for event data and JSON payloads.
- Accessibility: Keyboard navigable dashboard, screen reader support for event tables, ARIA labels on all interactive elements, high contrast mode support
- Voice & messaging: Technical/operational tone. Dashboard is for developers, operators, and platform administrators — not end users. Clear, jargon-appropriate labeling (e.g., "Circuit Breaker", "Policy Rule", "Event Envelope").
- Branding hooks: Vrooli color palette, "Central Nervous System" metaphor in onboarding, status badges for system health, pulsing indicators for live event streams

## Appendix
- Research: `scenarios/swarm-manager/research/vrooli-events-architecture/`
- Orchestration context: `scenarios/swarm-manager/.vrooli/initiatives/vrooli-events/orchestration-summary.md`
- Proto schemas: `packages/proto/schemas/vrooli-events/v1/`
- Generated types: `packages/proto/gen/go/vrooli-events/v1/domain/`
- Related initiatives: `vrooli-events` (this scenario), `notification-hub-greenfield` (primary consumer)
- Backlog items: `execute/vrooli-events-core-runtime`, `execute/discovery-event-emission-and-policy-cache`, `execute/vrooli-events-analytics-ui`
