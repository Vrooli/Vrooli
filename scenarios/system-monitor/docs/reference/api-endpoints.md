# API Endpoints Reference

**Base URL**: `http://localhost:${API_PORT}` (port assigned by lifecycle system, default 8080)

**Authentication**: None currently. All endpoints are publicly accessible.

**Runtime contract**: Proto-owned operations are served through generated
Connect procedure paths such as
`/vrooli.system_monitor.v1.metrics.MetricsService/GetCurrentMetrics`. The
`/api/v1/...` paths in the proto-owned sections below are HTTP annotation
inventory and historical compatibility context; they are not mounted as manual
REST routes after the bright-window cleanup. Runtime REST exceptions are health
probes, development pprof, logs, and forensics.

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
| GET | `/api/v1/metrics/pressure` | Typed Linux memory PSI/OOM snapshot; unavailable evidence is explicit |
| GET | `/api/v1/metrics/detailed` | Comprehensive metrics (CPU, memory, network, GPU, disk, processes, system health) |
| GET | `/api/v1/metrics/timeline` | Recent metrics timeline |
| GET | `/api/v1/metrics/processes` | Process monitoring data (zombies, high-thread, leak candidates) |
| GET | `/api/v1/metrics/processes/timeline` | Ranked process consumers over a time window, grouped by owner/scenario |
| GET | `/api/v1/forensics/processes` | Bounded CPU, RSS, or GPU-VRAM-ranked process attribution (`rank=cpu|rss|gpu`) |
| GET | `/api/v1/forensics/gpu` | Retained GPU utilization and VRAM timeline (`window=1h`) |
| GET | `/api/v1/forensics/pressure` | Retained memory PSI and OOM-kill timeline (`window=1h`) |
| GET | `/api/v1/metrics/infrastructure` | Infrastructure monitoring (DB pools, HTTP pools, queues, storage I/O) |
| GET | `/api/v1/metrics/disk` | Disk partition and usage detail through the generated Connect method |

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
| GET | `/api/v1/investigations/scripts` | List investigation scripts |
| GET | `/api/v1/investigations/scripts/{id}` | Get script by ID |
| POST | `/api/v1/investigations/scripts/{id}/execute` | Execute investigation script |

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

### Metrics lifecycle (retention & compaction)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/maintenance/metrics/retention/preview?days=<n>` | Read-only estimate of rows/bytes/time-range a prune would remove, with DB stats |
| POST | `/api/v1/maintenance/metrics/retention/apply` | Prune metrics older than the window. Body: `{"retentionDays": <n>, "confirm": true}`. Requires `confirm=true` |
| GET | `/api/v1/maintenance/metrics/compaction/preview` | Read-only DB stats and estimated reclaimable bytes |
| POST | `/api/v1/maintenance/metrics/compaction/apply` | Compact the DB (`VACUUM`). Body: `{"confirm": true}`. Requires `confirm=true` |

Destructive applies without `confirm=true` return `400` validation errors.
Compaction on a non-SQLite backend returns `503` (unsupported).

`[CODE: api/internal/handlers/maintenance.go]`

---

## Agent Manager status

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/agent/status` | Get agent status |

System Monitor does not select runners, models, or profile defaults. Its
scenario-owned `.vrooli/agent-profiles/default.json` declares portable
`roleRef` intent and is reconciled through Agent Manager. Operators inspect
role and native-resource state in Agent Manager; System Monitor only reports
its integration availability and active investigation count.

`[CODE: api/internal/handlers/investigations.go]`

---

## Missing Product Endpoints

The following endpoints are referenced by the UI but do not exist in the API:

- `POST /api/v1/processes/{pid}/kill` -- referenced by UI process kill dialog (silently fails)

Disk detail is implemented through `MetricsService.GetDiskDetail`
(`/vrooli.system_monitor.v1.metrics.MetricsService/GetDiskDetail`) and is
read-only. Its response may include storage-manager handoff notes when disk
pressure is high; system-monitor does not delete files or apply cleanup.

## Connect Migration Notes

Proto schemas and generated clients exist under `packages/proto/schemas/system-monitor/v1/` and `packages/proto/gen/`. The runtime mounts generated Connect handlers through `http.ServeMux`; gorilla/mux and proto-owned manual REST routes have been removed. Current non-blocking drift and REST exceptions are tracked in `[CODE: docs/internal/INTEROP_AUDIT.md]`.
