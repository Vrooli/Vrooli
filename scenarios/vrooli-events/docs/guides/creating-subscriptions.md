# Creating Subscriptions

Persistent subscriptions let you react to events from other scenarios without modifying the source. This is how notification-hub enables "get notified when a backlog item completes" without changing swarm-manager.

Implementation: [CODE: internal/subscription/subscription.go] | Store: [CODE: internal/subscription/sqlite.go] | Webhook delivery: [CODE: internal/subscription/webhook.go] | CRUD API: [CODE: api/handlers_subscription.go] | UI: [CODE: ui/src/pages/SubscriptionsPage.tsx], [CODE: ui/src/pages/SubscriptionHealthPage.tsx]

## Subscription Types

| Type | Use Case | Delivery |
|------|----------|----------|
| **Webhook** | Service-to-service integration | POST to a URL with retry and HMAC signature |
| **SSE** | Real-time UI or long-lived consumer | Reconnectable SSE stream with Last-Event-ID |

## Creating a Webhook Subscription

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{"name":"notify-on-completion","owner_scenario":"notification-hub","event_pattern":"swarm-manager.backlog.item-completed.v1","delivery_type":"webhook","delivery_target":"http://localhost:15200/api/v1/hooks/events","enabled":true}'
```

Every time a `swarm-manager.backlog.item-completed.v1` event is ingested, vrooli-events POSTs it to the target URL.

### Webhook Request Format

```http
POST /api/v1/hooks/events HTTP/1.1
Host: localhost:15200
Content-Type: application/json
X-VrooliEvents-Subscription: sub_abc123
X-VrooliEvents-Signature: sha256=abc123def456...

{"eventId":"swarm-manager.backlog.item-completed.v1","sourceScenario":"swarm-manager",...}
```

The HMAC-SHA256 signature lets receivers verify the event came from vrooli-events.

## Using Glob Patterns

Subscribe to broad categories using globs:

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/subscriptions" \
  -H 'Content-Type: application/json' \
  -d '{"name":"all-swarm","owner_scenario":"notification-hub","event_pattern":"swarm-manager.**","delivery_type":"webhook","delivery_target":"http://localhost:15200/api/v1/hooks/events","enabled":true}'

curl -X POST "http://localhost:${API_PORT}/api/v1/subscriptions" \
  -H 'Content-Type: application/json' \
  -d '{"name":"all-completions","owner_scenario":"notification-hub","event_pattern":"**.completed.v1","delivery_type":"webhook","delivery_target":"http://localhost:15200/api/v1/hooks/events","enabled":true}'

curl -X POST "http://localhost:${API_PORT}/api/v1/subscriptions" \
  -H 'Content-Type: application/json' \
  -d '{"name":"all-backlog","owner_scenario":"notification-hub","event_pattern":"*.backlog.*","delivery_type":"webhook","delivery_target":"http://localhost:15200/api/v1/hooks/events","enabled":true}'
```

## Testing a Subscription

Before relying on it in production, verify delivery works:

```bash
SUBSCRIPTION_ID=123
curl -X POST "http://localhost:${API_PORT}/api/v1/subscriptions/${SUBSCRIPTION_ID}/test"
```

This sends a synthetic test event matching the pattern and reports whether delivery succeeded.

## Monitoring Subscription Health

```bash
curl "http://localhost:${API_PORT}/api/v1/subscriptions/${SUBSCRIPTION_ID}/health"
```

Shows: total delivered, total failed, success rate, consecutive failures, and status (healthy/degraded/circuit_broken).

If consecutive failures exceed the threshold (default: 50), the subscription auto-disables. Re-enable it after fixing the target:

```bash
curl -X PUT "http://localhost:${API_PORT}/api/v1/subscriptions/${SUBSCRIPTION_ID}" \
  -H 'Content-Type: application/json' -d '{"enabled":true}'
```

## Example: Notification Hub Integration

notification-hub subscribes to events and routes them to devices:

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{"name":"backlog-complete-notify","owner_scenario":"notification-hub","event_pattern":"swarm-manager.backlog.item-completed.v1","delivery_type":"webhook","delivery_target":"http://localhost:15200/api/v1/hooks/events","enabled":true}'

curl -X POST http://localhost:${API_PORT}/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{"name":"policy-violation-notify","owner_scenario":"notification-hub","event_pattern":"vrooli-events.policy.violation.v1","delivery_type":"webhook","delivery_target":"http://localhost:15200/api/v1/hooks/events","enabled":true}'

curl -X POST http://localhost:${API_PORT}/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{"name":"needs-attention-notify","owner_scenario":"notification-hub","event_pattern":"**.needs-attention.*","delivery_type":"webhook","delivery_target":"http://localhost:15200/api/v1/hooks/events","enabled":true}'
```

notification-hub receives these webhooks, matches them against user-configured notification preferences, and delivers via ntfy (iPhone push), email, SMS, or webhook.
