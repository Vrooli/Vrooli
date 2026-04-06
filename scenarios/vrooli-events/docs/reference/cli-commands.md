# CLI Reference

The vrooli-events CLI is built on cli-core ScenarioApp with auto-discovery of the vrooli-events API.

## CLI Configuration

```bash
vrooli-events configure api_base http://localhost:15000/api/v1
vrooli-events status    # Show connection status and store health
```

## Event Commands

### query — Search stored events

```bash
vrooli-events query [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| --type | string | Glob pattern on event type (e.g., `swarm-manager.**`) |
| --source | string | Filter by source scenario |
| --correlation-id | string | Filter by correlation ID |
| --since | string | ISO-8601 or relative (e.g., `1h`, `24h`, `7d`) |
| --until | string | ISO-8601 or relative |
| --limit | int | Max results (default: 20) |
| --json | bool | Raw JSON output |

**Examples**:
```bash
vrooli-events query --type "swarm-manager.**" --since 1h --limit 10
vrooli-events query --correlation-id "trace-abc123" --json
vrooli-events query --source "agent-manager" --since 24h
```

### subscribe — Live SSE event stream

```bash
vrooli-events subscribe [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| --type | string | Glob pattern filter |
| --source | string | Source scenario filter |
| --target | string | Target scenario filter |
| --json | bool | Raw JSON output (one event per line) |

**Examples**:
```bash
vrooli-events subscribe --type "**"                    # All events
vrooli-events subscribe --type "swarm-manager.backlog.*"  # Backlog events
vrooli-events subscribe --source "agent-manager" --json    # Pipe-friendly
```

### stats — Store health and metrics

```bash
vrooli-events stats [--json]
```

Shows: total events, store size, oldest/newest event, subscriber counts, retention settings.

## Policy Commands

### policy list — List policy rules

```bash
vrooli-events policy list [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| --type | string | Filter: access, rate_limit, circuit_breaker |
| --source | string | Filter by source pattern |
| --target | string | Filter by target pattern |
| --json | bool | Raw JSON output |

### policy create — Create a policy rule

```bash
vrooli-events policy create --type <type> [flags]
```

**Access control**:
```bash
vrooli-events policy create --type access \
  --source "untrusted-scenario" \
  --target "agent-manager" \
  --effect deny \
  --priority 10
```

**Rate limiting**:
```bash
vrooli-events policy create --type rate_limit \
  --source "*" \
  --target "agent-manager" \
  --max-requests 100 \
  --window 60
```

**Circuit breaker**:
```bash
vrooli-events policy create --type circuit_breaker \
  --source "*" \
  --target "flaky-service" \
  --failure-threshold 5 \
  --cooldown 30
```

### policy update — Update a policy rule

```bash
vrooli-events policy update --id <id> [flags]
```

### policy delete — Delete a policy rule

```bash
vrooli-events policy delete --id <id>
```

### policy override — Manual circuit breaker override

```bash
vrooli-events policy override --id <id> --state closed [--ttl 3600]
```

### policy violations — Query violation log

```bash
vrooli-events policy violations [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| --source | string | Filter by source scenario |
| --target | string | Filter by target scenario |
| --type | string | Filter by rule type |
| --since | string | Time range start |
| --limit | int | Max results |
| --json | bool | Raw JSON output |

## Subscription Commands

### subscriptions list — List persistent subscriptions

```bash
vrooli-events subscriptions list [--owner <scenario>] [--json]
```

### subscriptions create — Create a subscription

```bash
vrooli-events subscriptions create \
  --name "backlog-notifications" \
  --pattern "swarm-manager.backlog.**" \
  --delivery-type webhook \
  --target "http://localhost:15200/api/v1/hooks/events"
```

| Flag | Type | Description |
|------|------|-------------|
| --name | string | Human-readable name (required) |
| --pattern | string | Glob pattern on event type (required) |
| --source-filter | string | Glob on source scenario |
| --delivery-type | string | `sse` or `webhook` (required) |
| --target | string | Webhook URL or SSE reconnect key (required) |
| --owner | string | Owner scenario name |

### subscriptions update — Update a subscription

```bash
vrooli-events subscriptions update --id <id> [flags]
```

### subscriptions delete — Delete a subscription

```bash
vrooli-events subscriptions delete --id <id>
```

### subscriptions health — View delivery health

```bash
vrooli-events subscriptions health --id <id> [--json]
```

### subscriptions test — Send test event

```bash
vrooli-events subscriptions test --id <id>
```

## Settings Commands

### retention — View or update retention settings

```bash
vrooli-events retention                              # View current settings
vrooli-events retention --retention-days 60 --max-size-gb 4  # Update
vrooli-events retention --prune-now                   # Trigger manual prune
```
