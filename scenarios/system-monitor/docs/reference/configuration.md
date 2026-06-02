# Configuration Reference

## Environment Variables

Set these in `.env` or export them before starting the scenario.

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `8080` | API server port |
| `UI_PORT` | `3003` | UI dashboard port |
| `DATABASE_URL` | `postgres://vrooli@localhost:5433/...` | PostgreSQL connection string |
| `QUESTDB_URL` | `http://localhost:9009` | QuestDB HTTP endpoint |
| `REDIS_URL` | `redis://localhost:6380` | Redis connection string |
| `ENABLE_CLAUDE_INVESTIGATIONS` | `true` | Enable AI-driven investigations via agent-manager |
| `CPU_WARNING_THRESHOLD` | `70` | CPU usage warning threshold (%) |
| `CPU_CRITICAL_THRESHOLD` | `90` | CPU usage critical threshold (%) |
| `MEMORY_WARNING_THRESHOLD` | `80` | Memory usage warning threshold (%) |
| `MEMORY_CRITICAL_THRESHOLD` | `95` | Memory usage critical threshold (%) |

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

## Agent-Manager Settings

Agent configuration controls how AI investigation agents are spawned:

| Setting | Description |
|---------|-------------|
| `runner` | Agent runner type (e.g., claude-code, codex) |
| `model` | AI model to use |
| `max_turns` | Maximum conversation turns |
| `timeout` | Agent execution timeout |
| `tools` | Enabled tools list |
| `skip_permissions` | Skip permission prompts |
| `requires_sandbox` | Run in sandbox mode |
| `requires_approval` | Require human approval before actions |

Manage via the API:

```bash
# Get current agent config
curl http://localhost:8080/api/v1/agent/config

# Update agent config
curl -X PUT http://localhost:8080/api/v1/agent/config \
  -H "Content-Type: application/json" \
  -d '{"runner": "claude-code", "model": "claude-sonnet-4-6"}'
```

## Storage Backend

The API defaults to **in-memory storage** for simplicity. Data is lost on restart.

To use PostgreSQL, set `DATABASE_URL` to a valid connection string. The schema is defined in `initialization/postgres/schema.sql`.

QuestDB can be configured via `QUESTDB_URL` for time-series metrics, but the API currently falls back to in-memory storage regardless.

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
