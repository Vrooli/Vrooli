# Quick Start

Get vrooli-events running and emit your first event in under 5 minutes.

## 1. Start the Scenario

```bash
cd scenarios/vrooli-events
make start
```

## 2. Verify Health

```bash
curl http://localhost:${API_PORT}/health | jq
```

You should see:

```json
{
  "status": "ok",
  "store": {
    "total_events": 0,
    "total_payload_bytes": 0
  },
  "subscribers": 0
}
```

## 3. Emit Your First Event

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "eventId": "my-scenario.test.hello.v1",
    "eventType": "test.hello.v1",
    "sourceScenario": "my-scenario",
    "payload": {"message": "Hello from vrooli-events!"}
  }'
```

Response: `202 Accepted`

## 4. Query Events

```bash
curl "http://localhost:${API_PORT}/api/v1/events?type=my-scenario.**" | jq
```

## 5. Subscribe to Live Events

In one terminal, start an SSE subscription:

```bash
curl -N "http://localhost:${API_PORT}/api/v1/events/subscribe?type=my-scenario.**"
```

In another terminal, emit another event — you'll see it arrive in the SSE stream in real-time.

## 6. Use the CLI

```bash
vrooli-events stats
vrooli-events query --limit 5
vrooli-events subscribe --type "**"
```

## What's Next

- [Architecture](concepts/ARCHITECTURE.md) — Understand the full system design
- [Integrating a Scenario](guides/integrating-a-scenario.md) — Add automatic event emission to your scenario
- [Creating Subscriptions](guides/creating-subscriptions.md) — React to events from other scenarios
- [Managing Policies](guides/managing-policies.md) — Set up access control, rate limits, and circuit breakers
- [API Reference](reference/api-endpoints.md) — Full endpoint documentation
