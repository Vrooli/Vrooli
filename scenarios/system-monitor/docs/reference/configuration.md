# Configuration Reference

## Environment Variables

Set these in `.env` or export them before starting the scenario.

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `8080` | API server port |
| `UI_PORT` | `3003` | UI dashboard port |
| `DATABASE_URL` | `postgres://vrooli@localhost:5433/...` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6380` | Redis connection string |
| `ENABLE_CLAUDE_INVESTIGATIONS` | `true` | Enable AI-driven investigations via agent-manager |
| `CPU_WARNING_THRESHOLD` | `70` | CPU usage warning threshold (%) |
| `CPU_CRITICAL_THRESHOLD` | `90` | CPU usage critical threshold (%) |
| `MEMORY_WARNING_THRESHOLD` | `80` | Memory usage warning threshold (%) |
| `MEMORY_CRITICAL_THRESHOLD` | `95` | Memory usage critical threshold (%) |
| `SYSTEM_MONITOR_PROC_SAMPLE_INTERVAL` | `20s` | Cadence of the per-process `/proc` sampler that feeds the attribution timeline. Go duration string. |
| `SYSTEM_MONITOR_PROC_SAMPLE_TOP_N` | `50` | Top-N processes retained independently by CPU, RSS, and GPU VRAM per sampling cycle; the bounded union is at most 3N rows. Dropped processes are logged, never silently capped. |
| `SYSTEM_MONITOR_RAW_RETENTION` | `6h` | How long raw per-process rows are kept before they are downsampled into per-minute rollups. Go duration string. |
| `SYSTEM_MONITOR_ROLLUP_RETENTION` | `720h` | How long per-owner/per-minute rollups are kept (default 30 days). Go duration string. |

`[CODE: api/internal/config/config.go]`

## Threshold Configuration

Thresholds determine when the system transitions between HEALTHY, WARNING, and CRITICAL states:

- **Warning**: CPU >= 70% or Memory >= 80%
- **Critical**: CPU >= 90% or Memory >= 95%

Thresholds can be adjusted via environment variables (above) or the Settings API:

```bash
# View current settings
curl http://localhost:8080/api/v1/settings

# Update thresholds
curl -X PUT http://localhost:8080/api/v1/settings \
  -H "Content-Type: application/json" \
  -d '{"cpu_warning_threshold": 75, "cpu_critical_threshold": 95}'
```

## Investigation Triggers

Auto-fix triggers are configured in `initialization/configuration/investigation-triggers.json`:

| Trigger | Threshold | Sustained Duration |
|---------|-----------|-------------------|
| High CPU Usage | 75% | 60s |
| Memory Pressure | 10% available | 30s |
| Low Disk Space | 90% used | 120s |
| Excessive Network Connections | 2000 connections | 30s |
| Process Anomaly | 25 processes | 10s |

Triggers can be managed via the API:

```bash
# List all triggers
curl http://localhost:8080/api/v1/investigations/triggers

# Update a trigger threshold
curl -X PUT http://localhost:8080/api/v1/investigations/triggers/{id}/threshold \
  -H "Content-Type: application/json" \
  -d '{"threshold": 80}'
```

`[CODE: initialization/configuration/investigation-triggers.json]`

## Agent-Manager integration

System Monitor's portable investigation profile is owned by
`.vrooli/agent-profiles/default.json`; it declares `roleRef` and runner-neutral
execution controls. System Monitor does not expose a runner/model/profile
editor. Reconcile the declared profile through Agent Manager and use Agent
Manager's role-policy and desired-permission surfaces for operator changes.

```bash
agent-manager profiles reconcile-scenario --scenario system-monitor --dry-run
```

## Storage Backend

The API defaults to **in-memory storage** for simplicity. Data is lost on restart.

To use PostgreSQL, set `DATABASE_URL` to a valid connection string. The schema is defined in `initialization/postgres/schema.sql`.

Persistent time-series storage is not configured yet; the API currently uses in-memory storage.

## Metrics Retention & Compaction

The metrics history is the dominant consumer of database size. Retention and
compaction settings bound that growth. They live in the settings file
(`initialization/configuration/system-monitor-settings.json`, canonical
`{version, metadata, settings}` shape) and are editable via the Settings API/CLI.

| Setting | Default | Description |
|---------|---------|-------------|
| `metrics_retention_days` | `30` | Metrics older than this are pruned by scheduled retention. |
| `retention_check_interval_seconds` | `3600` | Interval between scheduled retention runs (min 60). |
| `retention_run_on_startup` | `true` | Run retention once at startup so stale data is pruned without waiting a full interval. |
| `compact_after_retention` | `false` | When true, a scheduled retention prune is followed by database compaction. |

`RETENTION_DAYS` (env) only seeds the default when no settings file exists yet;
once the settings file is present, the settings values own behavior.

Retention runs on a settings-driven scheduler (not the storage layer). Changes
to the interval, window, or compaction policy take effect on the next cycle
without restarting the scenario. Scheduled retention runs regardless of the
monitor's active/inactive state because it is housekeeping, not collection.

## Per-Process Attribution Timeline

A single `/proc` walk per sampling cycle captures pid/ppid/cmdline/cwd/CPU%/RSS
for every live process, attributes each to its owning scenario (by matching
`.../scenarios/<name>/` in the working directory, parsing a `<scenario>-api`
binary name, and walking the parent chain so children inherit their launcher's
owner), and persists the bounded CPU/RSS top-N union to a `process_samples` table. Host
processes that belong to no scenario are bucketed as `unknown` (a first-class
result, not an error). This replaces the opaque `bash -c "ps | sort | head"`
pipelines with one cheap pass and turns the manual "top consumers by scenario"
forensic into a standing query.

Raw rows are kept for `SYSTEM_MONITOR_RAW_RETENTION`, then downsampled into
per-owner/per-minute rollups kept for `SYSTEM_MONITOR_ROLLUP_RETENTION`. Both
windows run on the same settings-driven scheduler as metrics retention.

Query the ranked timeline over a window, grouped by source scenario:

```bash
# REST
curl 'http://localhost:8080/api/v1/metrics/processes/timeline?window=5m&top=20'
curl 'http://localhost:8080/api/v1/metrics/processes/timeline?window=1h&owner=security-health'
curl 'http://localhost:8080/api/v1/metrics/pressure'
curl 'http://localhost:8080/api/v1/forensics/processes?window=1h&rank=rss&top=20'
curl 'http://localhost:8080/api/v1/forensics/processes?window=1h&rank=gpu&top=20'
curl 'http://localhost:8080/api/v1/forensics/gpu?window=1h'
curl 'http://localhost:8080/api/v1/forensics/pressure?window=1h'

# CLI
system-monitor metrics process-timeline --window 5m --top 20
system-monitor metrics process-timeline --owner security-health --json
```

### Manual maintenance

Retention deletes rows; compaction (`VACUUM`) reclaims the freed file space and
is serialized against metric writes. Both expose a read-only preview and a
destructive apply that requires explicit confirmation:

```bash
# Preview how many rows / bytes a 30-day window would prune
system-monitor maintenance retention preview --days 30

# Apply the prune (confirmation required)
system-monitor maintenance retention apply --days 30 --confirm

# Preview reclaimable space, then compact
system-monitor maintenance compact preview
system-monitor maintenance compact apply --confirm
```

See [operations/RUNBOOK.md](../operations/RUNBOOK.md) for the full safe workflow
and backup posture.

`[CODE: api/internal/services/maintenance.go, api/internal/services/retention_scheduler.go]`
