# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## Overview
- **Purpose**: Central event bus for inter-scenario communication. Provides durable event storage, real-time SSE-based pub/sub, and an analytics dashboard for monitoring event flow across the Vrooli ecosystem.
- **Primary users/verticals**: Other Vrooli scenarios (as event producers/consumers), platform operators (monitoring), developers (debugging event flows)
- **Deployment surfaces**: API (event ingestion + query + SSE streaming), CLI (query/subscribe/stats), UI (real-time analytics dashboard)
- **Value promise**: Enables event-driven architecture across scenarios with traceability, history, and real-time delivery. Every scenario can publish events and subscribe to events from other scenarios without point-to-point coupling.

## Operational Targets

### P0 -- Must ship for viability
- [x] OT-P0-001 | Event ingestion | POST /api/v1/events accepts EventEnvelope proto payloads, returns 202 Accepted, stores durably in SQLite WAL-mode
- [x] OT-P0-002 | SSE subscribe | GET /api/v1/events/subscribe delivers real-time events with glob-pattern filtering on event type, source, and target
- [x] OT-P0-003 | Event query | GET /api/v1/events returns stored events with filters (type glob, source, correlation_id, since, limit)
- [x] OT-P0-004 | Health endpoint | GET /health reports store stats, subscriber count, and overall status

### P1 -- Should have post-launch
- [x] OT-P1-001 | Glob-pattern matching | Structured event IDs ({scenario}.{domain}.{action}.{version}) with segment-aware * and ** glob patterns
- [x] OT-P1-002 | Correlation tracking | Events carry correlation_id for distributed trace grouping
- [x] OT-P1-003 | Dual-trigger pruning | Background goroutine every 6 hours: delete events older than 30 days OR when store exceeds 2GB
- [x] OT-P1-004 | Last-Event-ID resume | SSE clients can reconnect with Last-Event-ID header to replay missed events
- [x] OT-P1-005 | Backpressure handling | 64-capacity subscriber channels with drop+notify (dropped_count in heartbeat)

### P2 -- Future / expansion
- [ ] OT-P2-001 | Analytics dashboard | Real-time UI showing event throughput, top event types, subscriber status, store growth
- [ ] OT-P2-002 | CLI tools | Query, subscribe, and stats commands via ScenarioApp CLI
- [ ] OT-P2-003 | Policy enforcement | Access control, rate limiting, circuit breaking (separate initiative item)
- [ ] OT-P2-004 | Discovery integration | EmittingResolver decorator auto-publishes scenario discovery events

## Tech Direction Snapshot
- Preferred stacks / frameworks: Go (API + CLI), React + Vite + Tailwind (UI), SQLite with WAL mode (storage)
- Data + storage expectations: Embedded SQLite (no external database dependency). WAL mode for concurrent reads. Metadata table tracks total_payload_bytes for size-based pruning.
- Integration strategy: HTTP API for event ingestion, SSE for real-time delivery. Other scenarios integrate via direct HTTP calls to the events API. Proto-serialized payloads for type safety.
- Non-goals / guardrails: No authentication/authorization on endpoints (deferred). No policy CRUD API (separate item: vrooli-events-policy-api-and-middleware). No message queuing or guaranteed delivery (fire-and-forget with durable history). No clustering (single-node SQLite).

## Dependencies & Launch Plan
- Required resources: None (embedded SQLite, no external dependencies)
- Scenario dependencies: packages/proto (generated Go types for EventEnvelope, SubscriptionRequest, etc.)
- Operational risks: SQLite single-writer bottleneck at very high throughput; mitigated by WAL mode and async ingestion
- Launch sequencing: Core runtime (done) -> Policy API -> Discovery integration -> Analytics UI

## UX & Branding
- Look & feel: Dark theme primary, consistent with Vrooli platform aesthetics. Real-time charts and event stream visualization.
- Accessibility: Keyboard navigable dashboard, screen reader support for event tables
- Voice & messaging: Technical/operational tone. Dashboard is for developers and operators, not end users.
- Branding hooks: Vrooli color palette, monospace for event data, status badges for system health

## Appendix
- Research: `scenarios/swarm-manager/research/vrooli-events-architecture/`
- Proto schemas: `packages/proto/schemas/vrooli-events/v1/`
- Generated types: `packages/proto/gen/go/vrooli-events/v1/domain/`
