# Telemetry Endpoints

Endpoints for uploading and querying deployment telemetry.

## GET /telemetry

List telemetry summaries for all scenarios.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/telemetry"
```

**Response:**

```json
{
  "scenarios": [
    {
      "scenario": "picker-wheel",
      "total_events": 1523,
      "last_event": "2025-01-15T14:30:00Z",
      "failure_counts": {
        "migration_failed": 2,
        "dependency_unreachable": 5,
        "secrets_missing": 1
      },
      "recent_failures": [
        {
          "timestamp": "2025-01-15T14:25:00Z",
          "event": "dependency_unreachable",
          "details": { "service": "sqlite", "reason": "connection timeout" }
        }
      ]
    },
    {
      "scenario": "shared-auth",
      "total_events": 892,
      "last_event": "2025-01-15T14:28:00Z",
      "failure_counts": {},
      "recent_failures": []
    }
  ],
  "total_scenarios": 2
}
```

---

## POST /telemetry/upload

Upload telemetry events from deployed applications.

**Request (JSON array):**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/telemetry/upload" \
  -H "Content-Type: application/json" \
  -d '[
    {
      "scenario": "picker-wheel",
      "deployment_id": "deploy-123",
      "event": "app_start",
      "timestamp": "2025-01-15T12:00:00Z",
      "details": { "version": "1.0.0", "platform": "win-x64" }
    },
    {
      "scenario": "picker-wheel",
      "deployment_id": "deploy-123",
      "event": "ready",
      "timestamp": "2025-01-15T12:00:05Z",
      "details": { "startup_ms": 5000 }
    }
  ]'
```

**Request (JSONL - newline-delimited):**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/telemetry/upload" \
  -H "Content-Type: application/x-ndjson" \
  -d '{"scenario":"picker-wheel","event":"app_start","timestamp":"2025-01-15T12:00:00Z"}
{"scenario":"picker-wheel","event":"ready","timestamp":"2025-01-15T12:00:05Z"}'
```

**Event Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `scenario` | string | Yes | Scenario name |
| `deployment_id` | string | No | Deployment identifier |
| `event` | string | Yes | Event type |
| `timestamp` | string | No | ISO8601 timestamp (auto-added if missing) |
| `details` | object | No | Event-specific data |

**Response:**

```json
{
  "status": "accepted",
  "events_received": 2,
  "events_stored": 2
}
```

**Notes:**
- Maximum payload size: 4MB
- Events are appended to `{telemetry_dir}/{scenario}.jsonl`
- Missing timestamps are auto-filled with server time

---

## Standard Event Types

### Lifecycle Events

| Event | Description | Details |
|-------|-------------|---------|
| `app_start` | Application launched | `version`, `platform`, `mode` |
| `ready` | All services ready | `startup_ms`, `services` |
| `shutdown` | Clean shutdown | `reason`, `uptime_ms` |
| `crash` | Unexpected termination | `error`, `stack` |

### Service Events

| Event | Description | Details |
|-------|-------------|---------|
| `service_started` | Service process started | `service_id`, `pid`, `port` |
| `service_stopped` | Service stopped | `service_id`, `exit_code` |
| `service_crashed` | Service crashed | `service_id`, `error` |
| `dependency_unreachable` | Couldn't reach dependency | `service_id`, `dependency`, `reason` |

### Data Events

| Event | Description | Details |
|-------|-------------|---------|
| `migration_started` | Migration beginning | `version`, `direction` |
| `migration_completed` | Migration succeeded | `version`, `duration_ms` |
| `migration_failed` | Migration failed | `version`, `error` |

### Secret Events

| Event | Description | Details |
|-------|-------------|---------|
| `secrets_generated` | Per-install secrets created | `secret_ids` |
| `secrets_prompted` | User prompted for secrets | `secret_ids` |
| `secrets_missing` | Required secrets not provided | `secret_ids` |

### Update Events

| Event | Description | Details |
|-------|-------------|---------|
| `update_check_started` | Checking for updates | `channel`, `current_version` |
| `update_available` | Update found | `new_version`, `channel` |
| `update_downloaded` | Update downloaded | `version`, `size_bytes` |
| `update_installed` | Update applied | `version`, `restart_required` |
| `update_error` | Update failed | `error`, `stage` |

### Health Events

| Event | Description | Details |
|-------|-------------|---------|
| `health_check_failed` | Health check failed | `service_id`, `check_type`, `error` |
| `readiness_timeout` | Service didn't become ready | `service_id`, `timeout_ms` |

---

## Telemetry Storage

Events are stored in JSONL format:

```
{telemetry_dir}/{scenario}.jsonl
```

Default telemetry directory: `~/.vrooli/deployment/telemetry/`

Each line is a JSON object:

```jsonl
{"ts":"2025-01-15T12:00:00Z","scenario":"picker-wheel","event":"app_start","details":{"version":"1.0.0"}}
{"ts":"2025-01-15T12:00:05Z","scenario":"picker-wheel","event":"ready","details":{"startup_ms":5000}}
```

---

## Telemetry in Bundle Manifest

Configure telemetry in `bundle.json`:

```json
{
  "telemetry": {
    "file": "telemetry/deployment-telemetry.jsonl",
    "upload_to": "https://your-server/api/v1/telemetry/upload"
  }
}
```

- `file`: Local JSONL file path (relative to app data dir)
- `upload_to`: Optional endpoint for automatic upload

---

## Querying Telemetry

Use the CLI for filtered queries:

```bash
# All logs
deployment-manager logs <profile-id>

# Filter by level
deployment-manager logs <profile-id> --level error

# Search for term
deployment-manager logs <profile-id> --search "migration"

# Table format
deployment-manager logs <profile-id> --format table
```

---

## Related

- [CLI Deployment Commands](../cli/deployment-commands.md) - `logs` command
- [Desktop Workflow](../workflows/desktop-deployment.md) - Telemetry setup
- [Troubleshooting](../workflows/troubleshooting.md) - Using telemetry to debug
