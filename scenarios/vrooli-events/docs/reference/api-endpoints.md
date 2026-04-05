# API Reference

Base URL: `http://localhost:${API_PORT}/api/v1`

## Events

### POST /events — Ingest Event

Accepts an event and stores it durably. Returns immediately (202 Accepted) — storage is synchronous but the caller should treat this as fire-and-forget.

**Request Body** (JSON, proto-JSON compatible):

```json
{
  "eventId": "swarm-manager.backlog.item-completed.v1",
  "eventType": "backlog.item-completed.v1",
  "sourceScenario": "swarm-manager",
  "targetScenario": "notification-hub",
  "correlationId": "trace-abc123",
  "payload": {},
  "metadata": {}
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| eventId | string | yes | Structured event ID: `{scenario}.{domain}.{action}.{version}` |
| eventType | string | yes | Event type (typically the last 3 segments of eventId) |
| sourceScenario | string | yes | Scenario that produced this event |
| targetScenario | string | no | Intended target scenario (if directed) |
| correlationId | string | no | Trace grouping identifier |
| payload | object | no | Event-specific data (proto-JSON) |
| metadata | object | no | Additional context (user agent, IP, etc.) |

**Response**: `202 Accepted` with `{"id": <store_id>}`

### GET /events — Query Events

Returns stored events matching the given filters.

| Param | Type | Description |
|-------|------|-------------|
| type | string | Glob pattern on event type (e.g., `swarm-manager.**`) |
| source | string | Filter by source scenario |
| correlation_id | string | Filter by correlation ID |
| since | string | ISO-8601 timestamp lower bound |
| until | string | ISO-8601 timestamp upper bound |
| limit | int | Max results (default: 100, max: 1000) |
| offset | int | Pagination offset |

**Response**: `200 OK` with `{"events": [...], "total": N}`

### GET /events/subscribe — SSE Event Stream

Server-Sent Events stream of incoming events. Supports glob-pattern filtering and reconnection.

| Param | Type | Description |
|-------|------|-------------|
| type | string | Glob pattern filter on event type |
| source | string | Filter by source scenario |
| target | string | Filter by target scenario |

**Headers**:
- `Last-Event-ID`: Resume from this event store ID (replays missed events)

**SSE Format**:
```
id: 42
event: event
data: {"eventId":"swarm-manager.backlog.created.v1","sourceScenario":"swarm-manager",...}

:heartbeat {"subscribers":3,"dropped_count":0}

```

Heartbeat every 30s. Retry interval: 5000ms.

## Policies

### POST /policies — Create Policy Rule

**Request Body**:

```json
{
  "rule_type": "access",
  "source_scenario": "untrusted-scenario",
  "target_scenario": "agent-manager",
  "endpoint_pattern": "/api/v1/runs/**",
  "effect": "deny",
  "priority": 10,
  "enabled": true
}
```

Fields vary by `rule_type`:

| Field | access | rate_limit | circuit_breaker |
|-------|--------|------------|-----------------|
| source_scenario | glob | glob | glob |
| target_scenario | glob | glob | glob |
| endpoint_pattern | glob (optional) | glob (optional) | — |
| effect | allow\|deny | — | — |
| priority | int | — | — |
| max_requests | — | int | — |
| window_seconds | — | int | int |
| burst_allowance | — | int (optional) | — |
| failure_threshold | — | — | int |
| cooldown_seconds | — | — | int |
| success_threshold | — | — | int (default: 1) |

**Response**: `201 Created` with the created rule (including generated `id`)

### GET /policies — List Rules

| Param | Type | Description |
|-------|------|-------------|
| rule_type | string | Filter: access, rate_limit, circuit_breaker |
| source | string | Filter by source scenario pattern |
| target | string | Filter by target scenario pattern |
| enabled | bool | Filter by enabled state |

### GET /policies/:id — Get Rule

### PUT /policies/:id — Update Rule

Partial update — only provided fields are changed.

### DELETE /policies/:id — Delete Rule

### POST /policies/:id/override — Circuit Breaker Override

Force a circuit breaker into a specific state. Override expires after TTL (default: 1 hour).

**Request Body**:
```json
{
  "state": "closed",
  "ttl_seconds": 3600
}
```

### GET /policies/subscribe — SSE Policy Push

SSE stream of policy updates. Used by EmittingResolver and PolicyMiddleware to keep local caches current.

**SSE Events**:
- `policy_snapshot` — Full policy set (sent on initial connect)
- `policy_update` — Single rule created/updated/deleted
- `heartbeat` — Every 30s with `policy_version` counter

### GET /policies/violations — Query Violations

| Param | Type | Description |
|-------|------|-------------|
| source | string | Filter by source scenario |
| target | string | Filter by target scenario |
| rule_type | string | Filter by rule type |
| since | string | ISO-8601 lower bound |
| until | string | ISO-8601 upper bound |
| limit | int | Max results (default: 100) |

**Response**: `200 OK` with `{"violations": [...], "total": N}`

## Subscriptions

### POST /subscriptions — Create Subscription

**Request Body**:
```json
{
  "name": "backlog-notifications",
  "owner_scenario": "notification-hub",
  "event_pattern": "swarm-manager.backlog.**",
  "source_filter": "swarm-manager",
  "delivery_type": "webhook",
  "delivery_target": "http://localhost:15200/api/v1/hooks/events",
  "enabled": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Human-readable subscription name |
| owner_scenario | string | yes | Scenario that owns this subscription |
| event_pattern | string | yes | Glob pattern matching event types |
| source_filter | string | no | Glob pattern on source scenario |
| delivery_type | string | yes | `sse` or `webhook` |
| delivery_target | string | yes | Webhook URL or SSE reconnect key |
| enabled | bool | no | Default: true |

### GET /subscriptions — List Subscriptions

### GET /subscriptions/:id — Get Subscription

### PUT /subscriptions/:id — Update Subscription

### DELETE /subscriptions/:id — Delete Subscription

### GET /subscriptions/:id/health — Delivery Health

**Response**:
```json
{
  "total_delivered": 1234,
  "total_failed": 5,
  "success_rate": 0.996,
  "consecutive_failures": 0,
  "last_delivered_at": "2026-04-05T10:30:00Z",
  "last_failed_at": "2026-04-04T08:15:00Z",
  "status": "healthy"
}
```

Status values: `healthy`, `degraded` (>5% failure rate), `circuit_broken` (auto-disabled)

### POST /subscriptions/:id/test — Send Test Event

Sends a synthetic event matching the subscription's pattern and returns the delivery result.

**Response**: `200 OK` with `{"success": true, "delivery_time_ms": 45}` or `{"success": false, "error": "connection refused"}`

## Settings & Health

### GET /health — System Health

**Response**:
```json
{
  "status": "ok",
  "store": {
    "total_events": 15234,
    "total_payload_bytes": 45678912,
    "oldest_event": "2026-03-06T00:00:00Z",
    "newest_event": "2026-04-05T12:00:00Z"
  },
  "subscribers": {
    "event_sse": 3,
    "policy_sse": 12
  },
  "policy_cache": {
    "rules_loaded": 8,
    "last_updated": "2026-04-05T11:59:30Z"
  }
}
```

### GET /settings/retention — Get Retention Settings

### PUT /settings/retention — Update Retention Settings

**Request Body**:
```json
{
  "retention_days": 60,
  "max_size_bytes": 4294967296,
  "prune_interval_hours": 12
}
```

### POST /settings/retention/prune — Manual Prune

Triggers an immediate prune cycle. Returns count of events deleted.
