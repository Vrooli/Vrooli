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
vrooli-events subscriptions create \
  --name "notify-on-completion" \
  --pattern "swarm-manager.backlog.item-completed.v1" \
  --delivery-type webhook \
  --target "http://localhost:15200/api/v1/hooks/events"
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
# All swarm-manager events
vrooli-events subscriptions create --name "all-swarm" --pattern "swarm-manager.**" ...

# All completion events across all scenarios
vrooli-events subscriptions create --name "all-completions" --pattern "**.completed.v1" ...

# All backlog events from any scenario
vrooli-events subscriptions create --name "all-backlog" --pattern "*.backlog.*" ...
```

## Testing a Subscription

Before relying on it in production, verify delivery works:

```bash
vrooli-events subscriptions test --id <subscription-id>
```

This sends a synthetic test event matching the pattern and reports whether delivery succeeded.

## Monitoring Subscription Health

```bash
vrooli-events subscriptions health --id <subscription-id>
```

Shows: total delivered, total failed, success rate, consecutive failures, and status (healthy/degraded/circuit_broken).

If consecutive failures exceed the threshold (default: 50), the subscription auto-disables. Re-enable it after fixing the target:

```bash
vrooli-events subscriptions update --id <id> --enabled true
```

## Example: Notification Hub Integration

notification-hub subscribes to events and routes them to devices:

```bash
# Notify on backlog completions
vrooli-events subscriptions create \
  --name "backlog-complete-notify" \
  --pattern "swarm-manager.backlog.item-completed.v1" \
  --delivery-type webhook \
  --target "http://localhost:15200/api/v1/hooks/events"

# Notify on all policy violations
vrooli-events subscriptions create \
  --name "policy-violation-notify" \
  --pattern "vrooli-events.policy.violation.v1" \
  --delivery-type webhook \
  --target "http://localhost:15200/api/v1/hooks/events"

# Notify when any scenario needs attention
vrooli-events subscriptions create \
  --name "needs-attention-notify" \
  --pattern "**.needs-attention.*" \
  --delivery-type webhook \
  --target "http://localhost:15200/api/v1/hooks/events"
```

notification-hub receives these webhooks, matches them against user-configured notification preferences, and delivers via ntfy (iPhone push), email, SMS, or webhook.
