# Vrooli Events

The platform service for durable event receipts, real-time pub/sub, analytics, compliance auditing, and policy distribution across Vrooli.

Receipt observations use the canonical event type `vrooli.events.receipt.v1`.

## Why This Exists

Before vrooli-events, inter-scenario communication was invisible — scenario A called scenario B via HTTP, and nobody tracked it. There was no way to:

- Know which scenarios talk to each other, how often, or whether calls are failing
- Subscribe to events from other scenarios without modifying the source
- Enforce access control, rate limiting, or circuit breaking between scenarios
- Audit inter-scenario communication for compliance
- Build event-driven features (like notifications) that react to any scenario's activity

vrooli-events solves the shared-service part of this problem without becoming a mandatory runtime dependency. Standard clients use the local cache and receipt helpers in `packages/api-core/eventbus`; the complete behavioral contract is [Vrooli Events Platform Contract](../../docs/concepts/VROOLI_EVENTS_PLATFORM_CONTRACT.md).

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Scenario A (sender)                                            │
│                                                                  │
│  EmittingResolver                                                │
│  ├── Local policy cache ──→ Deny fast if policy forbids         │
│  ├── Resolve port (existing discovery logic)                     │
│  ├── Fire-and-forget event ──→ Background goroutine ──→ POST    │
│  └── HTTP call to Scenario B                                     │
└──────────────────────────────────────────────────┬──────────────┘
                                                   │
                    ┌──────────────────────────────┐│
                    │       vrooli-events           ││
                    │                               ││
                    │  POST /api/v1/events ────→ SQLite Store (WAL)
                    │       │                       │
                    │       ├──→ SSE Broker ────→ Event subscribers
                    │       └──→ Subscription ──→ Webhook delivery
                    │            engine                             │
                    │                               │
                    │  Policy engine                │
                    │  ├── Access control rules     │
                    │  ├── Rate limit counters      │
                    │  ├── Circuit breaker state    │
                    │  └── SSE policy push ────→ All scenario caches
                    │                               │
                    │  Pruner (background)          │
                    │  └── Retention + size cap     │
                    └──────────────────────────────┘│
                                                   │
┌──────────────────────────────────────────────────▼──────────────┐
│  Scenario B (receiver)                                           │
│                                                                  │
│  PolicyMiddleware                                                │
│  ├── Local policy cache ──→ Reject if policy forbids            │
│  └── Pass to handler                                             │
└─────────────────────────────────────────────────────────────────┘
                                                   │
                    ┌──────────────────────────────┘
                    ▼
┌─────────────────────────────────────────────────────────────────┐
│  notification-hub (subscriber)                                   │
│                                                                  │
│  Subscribes to vrooli-events via persistent subscriptions        │
│  Pattern: "swarm-manager.backlog.item-completed.v1"             │
│  Delivers to: iPhone (ntfy), email, SMS, webhook                │
└─────────────────────────────────────────────────────────────────┘
```

### Dual-End Policy Enforcement

Policy is enforced on **both** sides for defense-in-depth:

- **Sender side** (EmittingResolver in discovery package): Checks local policy cache before making the call. If denied, returns a `PolicyDeniedError` immediately — no network call made. Fast rejection, saves bandwidth.
- **Receiver side** (PolicyMiddleware in target scenario): Validates incoming requests against its own policy cache via `X-Source-Scenario` header. Even if a sender's cache is stale or a rogue caller bypasses discovery, the receiver rejects unauthorized requests.

Both caches are loaded at startup and kept current via SSE subscription to vrooli-events. If vrooli-events is unreachable, last-known policy holds (configurable fail-open or fail-closed per rule).

## Event ID Format

Events use structured IDs for precise subscription matching:

```
{scenario}.{domain}.{action}.{version}
```

Examples:
- `swarm-manager.backlog.item-completed.v1`
- `swarm-manager.backlog.needs-attention.v1`
- `deployment-manager.release.approved.v1`
- `git-control-tower.review.completed.v1`
- `vrooli-events.policy.violation.v1`

### Glob Pattern Matching

- `*` matches exactly one segment: `swarm-manager.*.created.v1`
- `**` matches one or more segments: `swarm-manager.**`
- Empty pattern matches everything

## Quick Start

```bash
cd scenarios/vrooli-events
make start    # Start via Vrooli lifecycle
make test     # Run tests
make logs     # View logs
make stop     # Stop
```

## API Endpoints

### Events

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/events | Ingest event (202 Accepted, fire-and-forget) |
| GET | /api/v1/events | Query events (filters: type, source, correlation_id, since, until, limit, offset) |
| GET | /api/v1/events/subscribe | SSE stream (filters: type, source, target; supports Last-Event-ID resume) |

### Policies

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/policies | Create policy rule (access control, rate limit, or circuit breaker) |
| GET | /api/v1/policies | List rules (filters: rule_type, source, target, enabled) |
| GET | /api/v1/policies/snapshot | Atomic enabled-rule snapshot for background client refresh |
| POST/GET | /api/v1/receipt-projections | Centrally managed safe receipt projection allow-lists |
| GET | /api/v1/policies/:id | Get rule by ID |
| PUT | /api/v1/policies/:id | Update rule |
| DELETE | /api/v1/policies/:id | Delete rule |
| POST | /api/v1/policies/:id/override | Manual circuit breaker override |
| GET | /api/v1/policies/subscribe | SSE policy push (real-time cache updates) |
| GET | /api/v1/policies/violations | Query policy violation log |

### Subscriptions

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/subscriptions | Create persistent subscription |
| GET | /api/v1/subscriptions | List subscriptions (filters: owner, pattern, enabled) |
| GET | /api/v1/subscriptions/:id | Get subscription |
| PUT | /api/v1/subscriptions/:id | Update subscription |
| DELETE | /api/v1/subscriptions/:id | Delete subscription |
| GET | /api/v1/subscriptions/:id/health | Subscription delivery health |
| POST | /api/v1/subscriptions/:id/test | Send test event to verify delivery |

### Settings & Health

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check (store stats, subscribers, policy cache status) |
| GET | /api/v1/settings/retention | Get retention settings |
| PUT | /api/v1/settings/retention | Update retention settings |
| POST | /api/v1/settings/retention/prune | Trigger manual prune |

## CLI

```bash
# Event operations
vrooli-events query --type "swarm-manager.**" --limit 10
vrooli-events query --correlation-id "abc123" --json
vrooli-events subscribe --type "*.completed.*"
vrooli-events stats

# Policy management
vrooli-events policy list
vrooli-events policy create --type access --source "swarm-manager" --target "agent-manager" --effect allow
vrooli-events policy create --type rate_limit --source "*" --target "agent-manager" --max-requests 100 --window 60
vrooli-events policy create --type circuit_breaker --source "*" --target "flaky-service" --failure-threshold 5 --cooldown 30
vrooli-events policy violations --since 1h

# Subscription management
vrooli-events subscriptions list
vrooli-events subscriptions create --name "backlog-notifications" --pattern "swarm-manager.backlog.**" --delivery-type webhook --target "http://localhost:15200/api/v1/hooks/events"
vrooli-events subscriptions health --id <id>
vrooli-events subscriptions test --id <id>

# Settings
vrooli-events configure api_base http://localhost:15000/api/v1
vrooli-events retention --retention-days 60 --max-size-gb 4
vrooli-events status
```

## Integration

### For scenario developers (automatic — no code changes)

Once the discovery package is updated with `EmittingResolver`, all existing inter-scenario calls automatically emit events. No changes to individual scenarios required.

### For scenarios that want receiver-side policy enforcement

```go
import "github.com/vrooli/api-core/discovery"

// Add to your HTTP router
router.Use(discovery.PolicyMiddleware("http://localhost:15000", "my-scenario"))
```

### For event consumers (like notification-hub)

Create a persistent subscription:

```bash
vrooli-events subscriptions create \
  --name "notify-on-completion" \
  --pattern "swarm-manager.backlog.item-completed.v1" \
  --delivery-type webhook \
  --target "http://localhost:15200/api/v1/hooks/events"
```

### Direct event publishing

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "eventId": "swarm-manager.backlog.created.v1",
    "eventType": "backlog.created.v1",
    "sourceScenario": "swarm-manager",
    "correlationId": "trace-abc123",
    "payload": {"item": "execute/my-feature", "status": "backlog"}
  }'
```

### Direct SSE subscription

```bash
curl -N "http://localhost:${API_PORT}/api/v1/events/subscribe?type=swarm-manager.**"
```

## Storage

Embedded SQLite with WAL mode — no external database required. Configurable retention (default: 30 days or 2GB, whichever triggers first). Policy rules, subscriptions, rate limit counters, circuit breaker state, and violation logs all stored in the same SQLite database.

## Related

- **Proto schemas**: `packages/proto/schemas/vrooli-events/v1/`
- **Discovery package**: `packages/api-core/discovery/` (integration target)
- **Orchestration context**: `scenarios/swarm-manager/.vrooli/initiatives/vrooli-events/orchestration-summary.md`
- **Primary consumer**: `scenarios/notification-hub/` (subscribes to events for push/email/SMS delivery)
- **PRD**: `scenarios/vrooli-events/PRD.md`
- **Requirements**: `scenarios/vrooli-events/requirements/`
