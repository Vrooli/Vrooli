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

## Testability Boundaries

| Component | Unit Test Approach | Integration Test Approach |
|-----------|--------------------|--------------------------|
| Event store | Mock SQLite (in-memory `:memory:`) | Real SQLite temp file |
| SSE broker | Mock subscriber channels | Real HTTP server + SSE client |
| Policy engine | Isolated evaluator with test rules | Full API → cache → evaluate cycle |
| Webhook delivery | Mock HTTP target server | Real target with test subscription |
| EmittingResolver | Mock events client, mock resolver | Real vrooli-events instance |
| PolicyMiddleware | Mock policy cache | Real SSE-fed cache |
| CLI | Mock API client | Real running scenario |

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
