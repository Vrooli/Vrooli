# API Endpoints Reference

**Base URL**: `http://localhost:${API_PORT}` (port assigned by lifecycle system, default 8080)

**Authentication**: None currently. All endpoints are publicly accessible.

---

## Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Basic health check |
| GET | `/api/v1/health` | Detailed health check with dependency status |

`[CODE: api/internal/handlers/health.go]`

---

## Metrics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/metrics/current` | Current system metrics snapshot |
| GET | `/api/v1/metrics/detailed` | Comprehensive metrics (CPU, memory, network, GPU, disk, processes, system health) |
| GET | `/api/v1/metrics/processes` | Process monitoring data (zombies, high-thread, leak candidates) |
| GET | `/api/v1/metrics/infrastructure` | Infrastructure monitoring (DB pools, HTTP pools, queues, storage I/O) |

### GET /api/v1/metrics/current

Query parameters:
- `fresh` (optional): Set to `1` or `true` for real-time collection instead of cached values

Response:
```json
{
  "cpu_usage": 45.2,
  "memory_usage": 62.1,
  "tcp_connections": 128,
  "gpu_usage": 0.0,
  "timestamp": "2026-02-18T10:30:00Z"
}
```

`[CODE: api/internal/handlers/metrics.go]`

---

## Investigations

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/investigations` | List investigations (query: `limit`, default 20) |
| GET | `/api/v1/investigations/latest` | Get latest investigation |
| GET | `/api/v1/investigations/{id}` | Get investigation by ID |
| POST | `/api/v1/investigations/trigger` | Trigger new investigation |
| POST | `/api/v1/investigations/agent/spawn` | Alias for trigger investigation |
| GET | `/api/v1/investigations/agent/current` | Get currently running agent status |
| GET | `/api/v1/investigations/agent/{id}/status` | Get agent status by investigation ID |
| POST | `/api/v1/investigations/agent/{id}/stop` | Stop agent for investigation |
| PUT | `/api/v1/investigations/{id}/status` | Update investigation status |
| PUT | `/api/v1/investigations/{id}/findings` | Update findings |
| PUT | `/api/v1/investigations/{id}/progress` | Update progress (0-100) |
| POST | `/api/v1/investigations/{id}/step` | Add investigation step |
| GET | `/api/v1/investigations/cooldown` | Get cooldown status |
| POST | `/api/v1/investigations/cooldown/reset` | Reset cooldown period |
| PUT | `/api/v1/investigations/cooldown/period` | Update cooldown duration |
| GET | `/api/v1/investigations/triggers` | Get all investigation triggers |
| PUT | `/api/v1/investigations/triggers/{id}` | Update trigger config |
| PUT | `/api/v1/investigations/triggers/{id}/threshold` | Update trigger threshold only |
| GET | `/api/v1/investigations/scripts` | List investigation scripts (placeholder -- returns empty array) |
| GET | `/api/v1/investigations/scripts/{id}` | Get script by ID (placeholder -- returns not found) |
| POST | `/api/v1/investigations/scripts/{id}/execute` | Execute investigation script (placeholder -- returns not found) |

### POST /api/v1/investigations/trigger

Request:
```json
{
  "auto_fix": true,
  "note": "High CPU detected"
}
```

Response:
```json
{
  "id": "inv-20260218-001",
  "status": "queued",
  "message": "Investigation triggered"
}
```

`[CODE: api/internal/handlers/investigations.go]`

---

## Reports

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/reports/generate` | Generate report |
| GET | `/api/v1/reports` | List all reports |
| GET | `/api/v1/reports/{id}` | Get report by ID |

### POST /api/v1/reports/generate

Request:
```json
{
  "type": "daily"
}
```

Valid types: `daily`, `weekly`

`[CODE: api/internal/handlers/reports.go]`

---

## Settings

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/settings` | Get all settings |
| PUT | `/api/v1/settings` | Update settings |
| POST | `/api/v1/settings/reset` | Reset to defaults |

`[CODE: api/internal/handlers/settings.go]`

---

## Maintenance

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/maintenance/state` | Get maintenance state |
| POST | `/api/v1/maintenance/state` | Set maintenance state (active/inactive) |

`[CODE: api/internal/handlers/settings.go]`

---

## Agent Configuration

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/agent/config` | Get agent configuration |
| PUT | `/api/v1/agent/config` | Update agent config |
| GET | `/api/v1/agent/runners` | Get available runners |
| GET | `/api/v1/agent/status` | Get agent status |

### PUT /api/v1/agent/config

Configurable fields: `runner`, `model`, `max_turns`, `timeout`, `tools`, `skip_permissions`, `requires_sandbox`, `requires_approval`

`[CODE: api/internal/handlers/investigations.go]`

---

## Tool Discovery Protocol

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tools` | Get tool manifest |
| GET | `/api/v1/tools/{name}` | Get specific tool definition |
| POST | `/api/v1/tools/execute` | Execute a tool |

`[CODE: api/internal/toolexecution/handler.go]`

---

## Missing Endpoints

The following endpoints are referenced by the UI but do not exist in the API:

- `GET /api/v1/metrics/timeline` -- referenced by UI sparkline charts
- `GET /api/v1/metrics/disk/details` -- referenced by UI disk detail view
- `POST /api/v1/processes/{pid}/kill` -- referenced by UI process kill dialog (silently fails)
