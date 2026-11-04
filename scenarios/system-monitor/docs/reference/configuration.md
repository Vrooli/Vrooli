# Configuration Reference

## Environment Variables

Set these in `.env` or export them before starting the scenario.

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `8080` | API server port |
| `UI_PORT` | `3003` | UI dashboard port |
| `ALERT_COOLDOWN_MINUTES` | `5` | Minimum gap between repeated point-in-time threshold records for the same metric. Condition alerts use their open/resolve lifecycle instead. |
| `SYSTEM_MONITOR_PROC_SAMPLE_INTERVAL` | `20s` | Cadence of the per-process `/proc` sampler that feeds the attribution timeline. Go duration string. |
| `SYSTEM_MONITOR_PROC_SAMPLE_TOP_N` | `50` | Top-N processes retained independently by CPU, RSS, and GPU VRAM per sampling cycle; the bounded union is at most 3N rows. Dropped processes are logged, never silently capped. |
| `SYSTEM_MONITOR_PROC_SAMPLE_OWNERS` | empty | Comma-separated list of up to 16 scenario owners whose CPU and RSS leaders are also retained each cycle, even when they fall outside the global top-N. Each configured owner adds at most 2N process rows. |
| `SYSTEM_MONITOR_RAW_RETENTION` | `6h` | How long raw per-process rows are kept before they are downsampled into per-minute rollups. Go duration string. |
| `SYSTEM_MONITOR_ROLLUP_RETENTION` | `720h` | How long per-owner/per-minute rollups are kept (default 30 days). Go duration string. |
| `SYSTEM_MONITOR_SOCKET_ATTRIBUTION_THRESHOLD` | `5000` | Established-TCP-connection count at or above which the network collector walks `/proc/<pid>/fd` to name the processes holding those sockets. Below it, only the aggregate count is collected. Set to `0` to disable attribution entirely. |

`[CODE: api/internal/config/config.go]`

## Threshold Configuration

Detection thresholds are persisted in `system-monitor-settings.json` and can be
changed in the System Settings modal or through the typed Settings API. The
saved values are the values read by the evaluator; these are not environment
seeds.

| Signal | Default warning / boundary | Default critical | Other controls |
|---|---:|---:|---|
| CPU usage | 85% | 97% | `alert_sustain_seconds`; CPU band history also uses the 92% high band, sample debounce, and cooldown settings. |
| Memory usage | 90% | 95% | `alert_sustain_seconds` |
| TCP connections | 5,000 | 20,000 | 10-minute growth window; warns on sustained leak-shaped growth from 500 connections after 5 samples, at 2x the opening count. |
| GPU usage | 90% | — | Unsupported when the host provides no GPU reading. |
| Metrics snapshot | — | 10 collection intervals stale | `metrics_snapshot_stale_intervals` |
| Orphaned agents | 1 hour old | 2 GiB known memory+swap, or reapable for 1 hour | `orphaned_agent_age_seconds`, `orphaned_agent_memory_bytes` |
| Memory pressure | elevated | critical | PSI and swap-in boundaries are separately configurable. |
| Disk pressure | 80% used | 95% used | 90% high band, debounce, cooldown, and fast-fill jump controls. |

`alert_sustain_seconds` defaults to 300 seconds. It controls how long a
condition must hold before opening, how long critical evidence must hold before
critical escalation, and how long the condition must stay clear before
resolution. Notification delivery preferences do not change detection.

Example API update:

```bash
# View current settings
curl http://localhost:8080/api/v1/settings

# Update thresholds
curl -X PUT http://localhost:8080/api/v1/settings \
  -H "Content-Type: application/json" \
  -d '{"cpu_threshold": 75, "cpu_critical_percent": 92, "tcp_connections_warning": 4000, "tcp_connections_critical": 15000, "alert_sustain_seconds": 180}'
```

## Notifications

System Monitor publishes `system_monitor.alert.notification.v1` domain events
to Vrooli Events. Notification Hub owns channel delivery and operator
preferences. The categories in `.vrooli/notifications.json` let operators
enable or mute each event family and set its minimum urgency:

Condition alerts notify on open, severity change, investigation escalation,
and resolution. Disk threshold transitions and storage-writer hot/cooled
transitions use their corresponding categories. Payloads carry a stable
dedupe key and alert correlation IDs; process and host details are marked
private. A notification publish failure is logged and does not roll back the
durable alert.

## Investigation behavior

After a condition is critical for the sustain duration, System Monitor records
an anomaly and runs its native catalog investigation. This is separate from
Notification Hub delivery settings. Agent Manager investigations remain an
explicit operator action; System Monitor does not automatically run an
AI-directed remediation.

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

The API uses durable SQLite storage by default. Set
`SYSTEM_MONITOR_STORAGE_MODE=memory` only for tests or explicit local
development. Memory mode loses history on restart and is rejected in
production. If SQLite cannot start, the API fails startup instead of silently
falling back to an unbounded in-memory history.

## Metrics Retention & Compaction

The metrics history is the dominant consumer of database size. Retention and
compaction settings bound that growth. They live in the settings file
(`api/internal/<domain>/configuration/system-monitor-settings.json`, canonical
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

## Collection Intervals and Thresholds

These live in the same settings file and are read live on each cycle.

| Setting | Default | Description |
|---------|---------|-------------|
| `metric_collection_interval` | `20` | Seconds between metric collection cycles. |
| `anomaly_detection_interval` | `30` | Seconds between anomaly detection passes. |
| `threshold_check_interval` | `20` | Seconds between disk-pressure and alert-rule evaluations (min 5). |
| `cooldown_period_seconds` | `300` | Minimum gap between automatically started native investigations (1–86,400 seconds). |
| `cpu_threshold` | `85.0` | CPU usage percentage treated as elevated; the `cpu_usage` condition warns at this value. |
| `cpu_high_percent` / `cpu_critical_percent` | `92.0` / `97.0` | CPU band-history boundaries; `cpu_critical_percent` is also the condition alert's critical boundary. |
| `memory_threshold` | `90.0` | Memory usage percentage treated as elevated; the `memory_usage` condition warns at this value. |

The CPU and memory critical boundaries are configurable independently. CPU
band history also uses `cpu_high_percent` and `cpu_critical_percent`; the
condition alert uses `cpu_threshold` and `cpu_critical_percent`.

## Alert Evaluation and Memory Pressure

The alert evaluator runs on `threshold_check_interval` and owns sustained
condition alerts. ThresholdScheduler owns disk band events and storage-writer
edges. Both publish through the same Notification Hub categories (see
[ARCHITECTURE.md § Alerting](../concepts/ARCHITECTURE.md#alerting)). All values
below are live; `0` selects the default where accepted by the Settings API.

| Setting | Default | Description |
|---------|---------|-------------|
| `alert_sustain_seconds` | `300` | A condition must hold this long before an alert opens, and stay clear this long before it resolves. Critical severity (and escalation) needs the critical condition itself to hold this long. |
| `memory_pressure_some_avg60_elevated_percent` | `10` | Linux: memory PSI `some avg60` at or above this is `elevated`. |
| `memory_pressure_full_avg60_critical_percent` | `5` | Linux: memory PSI `full avg60` at or above this is `critical`. |
| `memory_pressure_swap_in_elevated_bytes_per_second` | `10485760` (10 MiB/s) | Swap-in rate that is `elevated` on Linux, and that raises a macOS native level. |
| `memory_pressure_swap_in_critical_bytes_per_second` | `52428800` (50 MiB/s) | Swap-in rate that is `critical`. Must exceed the elevated rate. |
| `memory_critical_percent` | `95.0` | Memory usage boundary for a critical `memory_usage` condition. |
| `tcp_connections_warning` / `tcp_connections_critical` | `5000` / `20000` | TCP connection count boundaries; critical must exceed warning. |
| `tcp_growth_window_seconds` | `600` | History window used for leak-shaped connection growth. |
| `tcp_growth_floor` | `500` | Minimum connection count before a growth trend is meaningful. |
| `tcp_growth_min_samples` | `5` | Minimum samples required to classify a growth trend. |
| `tcp_growth_factor` | `2` | Required growth multiple across the configured window. |
| `gpu_usage_warning_percent` | `90` | GPU usage warning boundary. |
| `metrics_snapshot_stale_intervals` | `10` | Collection intervals without fresh metrics before a stale snapshot alert opens. |
| `orphaned_agent_age_seconds` | `3600` | Age before an orphaned or stuck-reapable agent session is considered actionable. |
| `orphaned_agent_memory_bytes` | `2147483648` (2 GiB) | Known memory plus swap held by orphaned sessions before critical severity. |
| `auto_investigate_critical_rules` | all supported rules enabled | Per-rule switches for starting the rule's native catalog investigation after a critical condition is sustained. These switches do not control alert detection or Notification Hub delivery. |

Notification Hub owns category enablement, urgency, and delivery. System
Settings owns detection thresholds, sustain timing, automatic investigation
policy, and investigation cooldown. The investigations page links to these
settings instead of maintaining a second copy of trigger thresholds.
The retired `investigation-triggers.json` file is ignored; it is not imported
or maintained. The legacy trigger endpoints return migration guidance.

`tcp_connections` can also warn on a sustained leak-shaped climb. A rule that
cannot observe its required platform evidence reports unsupported or failed;
missing evidence does not resolve an active condition.

## Disk Pressure Escalation

Disk usage is measured the way `df` measures it — `used / (used + available)`,
not `used / total`. The difference is the superuser reserve, which was 93 GB on
the host during the 2026-07-31 incident: reading free blocks reported 87 percent
while `df` reported 93, keeping every safeguard below its critical threshold
while the filesystem was unwritable for the runtime supervisor.

| Setting | Default | Description |
|---------|---------|-------------|
| `disk_threshold` | `80.0` | The **warning** band boundary: usage at which pressure starts being recorded. There is deliberately no separate `disk_warning_percent`; two settings for one boundary is how a configuration surface drifts from the code reading it. |
| `disk_high_percent` | `90.0` | The **high** band: start bounded storage-manager recovery. |
| `disk_critical_percent` | `95.0` | The **critical** band: storage-manager applies safe/regenerable providers with no operator present. |
| `disk_escalation_cooldown_seconds` | `1800` | Minimum gap between two records for the same band. Without it a disk parked above a boundary alerts every tick. |
| `disk_escalation_debounce_ticks` | `2` | Consecutive observations a new band needs before it takes effect, so one noisy sample cannot escalate. |
| `disk_fast_fill_jump_percent` | `5.0` | A rise of at least this many points in one evaluation escalates immediately, bounding the delay debounce introduces. |

The three band boundaries must ascend
(`disk_threshold` < `disk_high_percent` < `disk_critical_percent`). The Settings
API rejects a write that does not, so an operator learns immediately rather than
believing bands are in force that are not. A settings *file* that violates the
order is repaired to defaults on load instead, because refusing to load would
take the service down.

De-escalation is not debounced and resets the cooldown: dropping below a
boundary is good news, and holding a stale high band would keep remediation
armed after the pressure is gone.

Read current pressure, the active band, the last violation, and the last
remediation result with:

```bash
curl -s http://localhost:16914/api/v1/disk-pressure
```

Escalation flows to storage-manager, which runs only `safe`-tier providers
unattended. See
[../operations/RUNBOOK.md](../operations/RUNBOOK.md#disk-pressure-detection-and-escalation).

## Per-Process Attribution Timeline

A single `/proc` walk per sampling cycle captures pid/ppid/cmdline/cwd/CPU%/RSS
for every live process, attributes each to its owning scenario (by matching
`.../scenarios/<name>/` in the working directory, parsing a `<scenario>-api`
binary name, and walking the parent chain so children inherit their launcher's
owner), and persists the bounded global CPU/RSS/GPU leaders plus the configured
owners' CPU/RSS leaders to a `process_samples` table. Host processes that
belong to no scenario are bucketed as `unknown` (a first-class result, not an
error). This replaces the opaque `bash -c "ps | sort | head"` pipelines with
one cheap pass and turns the manual "top consumers by scenario" forensic into
a standing query.

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
curl 'http://localhost:8080/api/v1/forensics/processes?start_utc=2026-11-03T03:49:15Z&end_utc=2026-11-03T03:49:48Z&owner=device-control&rank=cpu_seconds'
curl 'http://localhost:8080/api/v1/forensics/process-samples?start_utc=2026-11-03T03:49:15Z&end_utc=2026-11-03T03:49:48Z&owner=device-control&limit=1000'
curl 'http://localhost:8080/api/v1/forensics/gpu?window=1h'
curl 'http://localhost:8080/api/v1/forensics/pressure?window=1h'

# CLI
system-monitor metrics processes --json
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
