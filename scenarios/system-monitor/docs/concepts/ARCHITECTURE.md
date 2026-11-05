# Architecture

## Purpose

System Monitor provides real-time server monitoring with threshold-based anomaly detection, AI-driven investigation via agent-manager, and automated reporting. It collects metrics from 6 pluggable collectors, evaluates configurable thresholds, and spawns AI agents to investigate anomalies. Its ecosystem role is a programmatic observability capability: other scenarios should be able to call typed health, metrics, investigation, report, settings, maintenance, capacity, and scripts surfaces without scraping logs or hand-parsing JSON.

## Component Diagram

```
+---------------------------------------------------------+
|  UI (React + Vite)         |  CLI (Go)                  |
|  Connect JSON polling      |  generated Connect clients |
+------------+---------------+-----------+----------------+
             |                           |
             v                           v
+---------------------------------------------------------+
|  API (Go)                                               |
|                                                         |
|  Connect handlers / REST exceptions --> services/ -> repo |
|                                                         |
|  collectors/  |  agentmanager/  |  services/            |
|  middleware/  |  server/        |  config/               |
+---------+--------+--------+--------+---+----------------+
          |        |        |        |   |
          v        v        v        v   v
      Postgres  Redis  Ollama  agent-manager
```

## Machine selection and remote presence

The header's machine selector sets the subject of the whole monitoring view.
The picker, identity strip, and presence note derive reachability, heartbeat age,
and granted actions from the same machine facts. A silent registered machine
stays visible; it does not disappear as though it had been deleted.

For a remote subject, local incident history, alerts, infrastructure, and
investigations must not appear to describe that subject. Their panels explain
the local-only boundary. A stale or unreachable reading carries its observed
reason and last-success information; reconnect messaging uses the actual retry
loop rather than promising an invented retry interval.

[CODE: ui/src/features/machines/presence.ts]
[CODE: ui/src/features/machines/components/MachinePicker.tsx]
[CODE: ui/src/features/machines/components/MachineIdentityStrip.tsx]
[CODE: ui/src/features/machines/components/MachinePresenceNote.tsx]
[CODE: ui/src/features/machines/components/LocalOnlyPanels.tsx]

These constraints preserve the accepted machine-linking design. Its dated
proposal and screenshots are historical evidence, recoverable through the
[project preservation record](../../../../docs/internal/PROGRESS.md#documentation-cleanup--2026-09-07).

## Layer Architecture

The API follows clean architecture with three main layers:

### Handlers (Presentation)
Connect-RPC is the transport for proto-owned operations. The runtime mounts generated Connect handlers on the standard library `http.ServeMux`; gorilla/mux and proto-owned manual REST routes have been removed. REST remains only for explicit exceptions: health probes, development pprof, and raw logs/forensics. Handlers depend on narrow interfaces (`MonitorQuerier`, `InvestigationManager`, `ScriptRunner`, `ReportGenerator`, `SettingsProvider`) defined in `handlers/interfaces.go`, keeping them decoupled from concrete service types.

`[CODE: api/internal/handlers/]`

### Services (Business Logic)
Coordination and orchestration layer. Five core services:
- **MonitorService** -- metric collection, threshold evaluation, system health
- **InvestigationService** -- investigation lifecycle, agent orchestration, cooldown
- **AlertService** -- threshold alerts, webhook notifications, cooldown management
- **ReportService** -- daily/weekly report generation
- **SettingsManager** -- configuration management

All dependencies flow through interfaces (`MetricsSource`, `AgentExecutor`, `HTTPDoer`, `Clock`, `ConfigStore`, `CommandRunner`), enabling isolated unit testing.

`[CODE: api/internal/services/]`

### Repository (Persistence)
Storage abstraction via `MetricRepository`, `InvestigationRepository`, `AlertRepository`, `ReportRepository` interfaces. SQLite is the durable runtime backend. The in-memory backend is explicit test/development configuration only.

`[CODE: api/internal/repository/]`

## Integration Seams

### Proto and CLI Contract
Proto schemas live in `packages/proto/schemas/system-monitor/v1/` and generated clients live under `packages/proto/gen/`. `cli/manifest.json` is the CLI command contract and names the authoritative Connect RPC for every proto-backed verb. The CLI uses generated Connect clients for scenario-domain commands, while omitted proto methods are documented in the manifest with reasons.

`[CODE: packages/proto/schemas/system-monitor/v1/]`
`[CODE: cli/manifest.json]`

### Collector Interface
Six collectors (CPU, memory, disk, network, process, GPU) implement a common interface. New collectors are added by creating a file in `collectors/` without touching existing code. `MonitorService` accepts `WithCollectors(...)` for test injection. Collectors should be efficient by construction: steady-state collection should use native `/proc`/syscall reads instead of shell pipelines, keep the normal cycle near zero forks, share a single process walk between process-health metrics and attribution sampling, and expose self-metrics for cycle duration and fork deltas.

`[CODE: api/internal/collectors/]`

### Agent-Manager Client
HTTP client for the agent-manager API. The `AgentExecutor` interface abstracts all agent-manager operations used by `InvestigationService`.

`[CODE: api/internal/agentmanager/]`

### Server Infrastructure
Router setup, middleware wiring, graceful shutdown, database initialization, and structured logging.

`[CODE: api/internal/server/]`

### Clock and ConfigStore
`Clock` interface abstracts time operations for deterministic testing. `ConfigStore` interface abstracts configuration file I/O. Both are injected via functional options.

## Memory pressure

Memory usage percent is not a pressure signal. On 2026-10-07 this host read a
"healthy" 53% memory while swap held 62 of 68 GB and swap-in ran at
5,000-12,000 pages/s; nothing alerted. The pressure collector therefore
publishes a platform-neutral `memory_pressure` reading built from stall and
paging evidence, exposed as `DetailedMetrics.memory_details.memory_pressure`
and judged by the `memory_pressure` alert rule.

The reading has a `level` (`normal`, `elevated`, `critical`, `unknown`), the
`level_source` and `level_reason` that produced it, and every signal in the
contract below. Each signal carries `measured`, `unsupported`, `failed`, or
`not_yet_sampled` plus provenance; a signal a platform cannot measure is
present with a reason and is never reported as zero. `unknown` means no
supported signal could produce a verdict and is never treated as healthy.

| Signal | Linux | macOS | Windows |
|--------|-------|-------|---------|
| `psi_{some,full}_avg{10,60,300}`, `psi_{some,full}_total_us` | `/proc/pressure/memory` | unsupported | unsupported |
| `swap_in_bytes_per_second`, `swap_out_bytes_per_second` | `/proc/vmstat` `pswpin`/`pswpout` × page size | `host_statistics64` `swapins`/`swapouts` × page size | unsupported: Pages Input/sec mixes pagefile and mapped-file reads |
| `major_faults_per_second` | `/proc/vmstat` `pgmajfault` | unsupported (no host-wide counter) | unsupported |
| `page_ins_per_second` | unsupported | `host_statistics64` `pageins` | PDH `\Memory\Pages Input/sec` (includes memory-mapped file reads) |
| `page_reads_per_second` | unsupported | unsupported | PDH `\Memory\Page Reads/sec` |
| `compressions_per_second`, `decompressions_per_second` | unsupported (zswap not collected) | `host_statistics64` | unsupported |
| `paging_file_usage_percent` | unsupported | unsupported | PDH `\Paging File(_Total)\% Usage` |
| `commit_charge_percent` | unsupported | unsupported | `GlobalMemoryStatusEx` (TotalPageFile − AvailPageFile) / TotalPageFile |
| `native_pressure_level` | unsupported | `kern.memorystatus_vm_pressure_level` (1, 2, 4) | `QueryMemoryResourceNotification(LowMemoryResourceNotification)` (0, 1) |

Level derivation (`collectors.DeriveMemoryPressureLevel`):

- **Linux** — PSI decides: `critical` when `full avg60` ≥ 5% or swap-in ≥ 50 MiB/s;
  `elevated` when `some avg60` ≥ 10% or swap-in ≥ 10 MiB/s. Without PSI the
  swap-in rate alone decides; with neither the level is `unknown`.
- **macOS** — the kernel's own verdict is authoritative (1 → normal,
  2 warn → elevated, 4 → critical); the swap-in boundaries may raise it, never
  lower it. Without the native level, swap-in alone decides.
- **Windows** — the low-memory notification is the only native verdict
  (signaled → critical, otherwise normal). Windows has no native elevated band,
  and Pages Input/sec is deliberately not used for the level because it counts
  memory-mapped file reads.

Boundaries are settings (`memory_pressure_*`, see
[configuration](../reference/configuration.md#alert-evaluation-and-memory-pressure)).
The memory detail view shows this server-derived verdict, its determining
reason, observation time, directional swap rates and 60-second PSI stalls.
Combined swap traffic and swap occupancy remain separate observations: neither
alone proves thrashing. Missing or failed signal states retain their reasons.

Rates come from the pressure collector's shared counter-rate tracker, so the
first sample after start is `not_yet_sampled` on every platform. "Sustained"
is the alert evaluator's job, not the collector's.

**Kernel allocation accounting.** `kernel_memory` is a separate typed object in
current metrics, detailed memory and the retained metrics timeline. Linux uses
shared host-inventory `/proc/meminfo` fields `Slab`, `SReclaimable` and
`SUnreclaim`, converted from kB to bytes, plus physical RAM for context. Each
field carries its own measured/unavailable state and source observation time.
Other platforms report unsupported; missing fields remain absent measurements,
acquisition failures remain failed, and historical rows predating collection
report not-yet-sampled rather than zero. Existing JSON cycle retention stores
these gauges without a schema migration. The memory detail view plots the
retained amounts and shows unreclaimable slab as a share of physical RAM.

These are allocation gauges, not a pressure or leak verdict. Correlate sustained
growth with workload, stalls and kernel-cache attribution before diagnosing a
leak. The collector adds no privileged `/proc/slabinfo` probe, automatic cleanup
or threshold. Non-slab kernel allocations are outside this object.
The native `memory-pressure` investigation includes the latest scheduler-owned
kernel accounting alongside the pressure reading, preserving each field's
availability and source time. A missing or failed kernel source is reported
explicitly without discarding the pressure result. This is an observation of
aggregate accounting; it does not identify a cache, allocator, or leak.

The metrics timeline uses the requested interval to select the newest cycle in
each time bucket. SQLite selects bucket candidates with bounded indexed seeks;
the memory repository applies the same selection to its retained cycles. The
response covers at most the newest 1,000 buckets and reports the resulting
effective window when a finer interval would exceed that limit. Empty buckets
do not imply zero or continuous coverage.

The native `process-read-activity` investigation is on demand. Linux reads at
most 4,096 processes twice, separated by two seconds, and ranks at most 20
matched processes by bytes returned by read-like calls per second, plus a
separate top 20 by storage-layer bytes. Rates use each process's actual observation
interval and require matching process start identities and nondecreasing
counters. The result reports permission failures, exits, unmatched processes
and truncation. Linux `/proc/<pid>/io` includes waited-for children, so rows can
overlap; short-lived processes between observations can be missed. Scenario
owners are heuristic. Other platforms return `unsupported`; no rate is
invented. This investigation does not run on the regular collection cycle.

**Attribution.** At `elevated` or `critical` the monitor attaches up to five
contributors from the shared process sampler, ranked by measured fault rate and
then swapped memory, each with the scenario owner the attributor computes:

| Platform | Fault counter | Swapped memory |
|----------|---------------|----------------|
| Linux | `/proc/<pid>/stat` `majflt` | `/proc/<pid>/status` `VmSwap` |
| macOS | `proc_pid_rusage` `ri_pageins`; other users' processes return EPERM and are reported unsupported | unsupported (no per-process swap accounting) |
| Windows | `PROCESS_MEMORY_COUNTERS.PageFaultCount` (hard **and** soft faults) | `PagefileUsage` (private commit) |

**macOS binding.** `golang.org/x/sys` exposes no Mach calls and cgo is not
used. `api/internal/darwinsys` calls `mach_host_self`, `host_statistics64`,
and `proc_pid_rusage` through libSystem trampolines (the technique x/sys uses
for every darwin syscall). Verified at runtime on macOS 15.7 Intel against
`vm_stat`; Apple Silicon and all Windows paths are compile- and fixture-verified
only.

[CODE: api/internal/collectors/memory_pressure.go]
[CODE: api/internal/collectors/pressure.go]
[CODE: api/internal/collectors/platform_paging_darwin.go]
[CODE: api/internal/collectors/platform_paging_windows.go]
[CODE: api/internal/darwinsys/]
[CODE: api/internal/services/memory_pressure.go]

## Alerting

Alerts are server state. `services.AlertEvaluator` runs on
`threshold_check_interval`, evaluates every rule against the shared observation
snapshot, and persists sustained `condition` alerts. ThresholdScheduler owns
disk band records and storage-writer edge events. The CLI (`status`, `alerts`,
`watch`) and the dashboard alert panel read `AlertsService.ListAlerts`; no
client derives an alert from a metric value.

| Rule | Condition |
|------|-----------|
| `metrics_snapshot` | critical when no snapshot exists or it is older than `metrics_snapshot_stale_intervals` collection intervals (default 10) |
| `cpu_usage` | warning ≥ `cpu_threshold` (default 85%), critical ≥ `cpu_critical_percent` (default 97%) |
| `memory_usage` | warning ≥ `memory_threshold` (default 90%), critical ≥ `memory_critical_percent` (default 95%) |
| `tcp_connections` | configurable warning/critical counts (defaults 5,000 / 20,000), plus warning on a leak-shaped climb using configured window, floor, sample count, and growth factor. Hundreds of connections are normal on this host, so there is no low fixed level. |
| `gpu_usage` | warning ≥ `gpu_usage_warning_percent` (default 90%; unsupported without a GPU reading) |
| `memory_pressure` | elevated → warning, critical → critical, unknown → rule unsupported |
| `orphaned_agents` | reads `vrooli agent list --json` (`state`, `kernel_state`, `owner{kind,ref,…}`, `memory_bytes`, `swap_bytes`, `reapable`, `reap_reason`). Warning when an orphan exceeds `orphaned_agent_age_seconds` (default 1 h); critical when known memory+swap exceeds `orphaned_agent_memory_bytes` (default 2 GiB) or a reapable row remains so for the configured age. Null or absent memory is reported unknown, never 0. Unsupported ("control plane does not classify orphans yet") when rows carry no `owner`. |

The CPU, memory, and GPU boundaries are the ones the CLI used to apply
client-side; the TCP rule is the dashboard's former card rule. Dashboard card
badges count active server alerts per rule; no client derives an alert.

Lifecycle:

- A condition must hold continuously for `alert_sustain_seconds` (default 300)
  before an alert opens; there is at most one open alert per rule and key.
- An open alert becomes critical only after the critical condition itself has
  held for the sustain duration, and returns to warning when it falls back.
- It resolves after the condition has stayed clear for the sustain duration;
  a relapse restarts that clock.
- A rule that is unsupported or failed neither opens nor resolves anything,
  and its timers restart when evidence returns. Rule states are returned with
  every `ListAlerts` response.
- Open alerts are re-adopted from storage on restart, so dedup survives it.
- A paused monitor evaluates nothing and leaves open alerts as they are.

**Escalation.** Once per open alert, a sustained critical condition is handed
to `AnomalyEscalator`: it records an `anomaly` event and runs the rule's native
catalog investigation (`memory-pressure` for memory rules, otherwise
`resource-leak-detector`), persisting the run with the anomaly id and a
`sustained_<rule>` finding, where run-history readers such as the infra-health
runtime scanner find it. It publishes a distinct `alert.investigation`
notification category. It never spawns an agent-manager investigation;
that remains an operator action. System Monitor never kills processes.
`auto_investigate_critical_rules` in System Settings controls this per rule;
disabling it leaves detection and alert notifications active while suppressing
the native investigation. `cooldown_period_seconds` is the shared minimum gap
for automatic investigations. Notification Hub separately controls category
delivery and urgency, so muting a category does not change detection or policy.

**Events.** Disk/CPU band history from the threshold scheduler and
escalation anomalies are point-in-time records (`threshold_violation`,
`anomaly`) created resolved. They keep history and notifications without
leaving rows that never close; `ListAlerts` includes them only with
`include_events`. Rows written before this change remain open in storage and
are excluded from the default view by type.

**Operator notifications.** Persisted alert lifecycle transitions and
storage-writer hot/cooled edges publish the generic
`system_monitor.alert.notification.v1` event. Category keys come from
`.vrooli/notifications.json`; Notification Hub owns per-category enablement,
minimum urgency, and delivery. System Monitor owns detection thresholds and
alert lifecycle. Muting a category does not pause detection or investigation.

**Legacy open violations.** 1,745 `threshold_violation` rows written before
this change have no `resolved_at`. The repository has no migration mechanism
(the schema is declarative and idempotent), so none was added. If they are to
be closed, the intended one-time operation is: resolve `type =
'threshold_violation'` rows with `resolved_at IS NULL` and `timestamp` earlier
than the evaluator's first start, recording the reason "legacy one-off event".
It is idempotent by construction and should be owned by the scenario's storage
lifecycle when a migration seam exists.

[CODE: api/internal/services/alert_evaluator.go]
[CODE: api/internal/services/alert_rules.go]
[CODE: api/internal/services/alert_escalation.go]
[CODE: api/internal/services/orphan_agents.go]
[CODE: api/internal/handlers/alerts.go]

## Data Flow

```
Metric Collectors (CPU, memory, disk, network, process, GPU)
        |
        v
MonitorService.collectAll()
        +--> metrics history
        |
        +--> AlertEvaluator (sustain, dedup, resolve)
        |        +--> AlertRepository
        |        +--> Vrooli Events --> Notification Hub preferences and delivery
        |        +--> sustained critical --> AnomalyEscalator --> native catalog run
        |
        +--> ThresholdScheduler (disk remediation, CPU band history,
                                 storage-writer edges) --> durable events
                                                     +--> notifications
```

## Key Design Decisions

- **In-memory default**: Keeps startup simple; persistent metric history remains future work
- **HTTP polling over WebSocket**: Simpler architecture, slightly higher latency (UI polls at 5s/60s/4s intervals)
- **Connect-first API contract**: Proto schemas and generated clients are the durable interface. REST remains only for explicit exceptions: ops probes, raw logs/forensics, third-party/webhook shapes, protocol-specific tool surfaces, and multipart/blob traffic.
- **Go CLI, manifest as SSOT**: The CLI command surface is declared in `cli/manifest.json`; commands use generated Connect clients and keep bindings aligned with proto descriptors.
- **Native collector path**: Host collection should prefer `/proc` and syscalls over `bash -c` pipelines. Shell execution remains behind seams for exceptional cases and tests should be able to assert the steady path is fork-free.
- **Aligned scheduler target**: Metrics, thresholds, anomaly checks, health probes, and process attribution should converge toward aligned intervals so system-monitor's own overhead is measurable and bounded.
- **Interface-driven services**: All service dependencies flow through interfaces for testability
- **Pluggable collectors**: New metric sources added without modifying existing code

## Metrics maintenance

The retention scheduler prunes the `metrics` table hourly. Every collector keeps
its rows for `metrics_retention_days` (default 30), except collectors with a
shorter window in `SYSTEM_MONITOR_COLLECTOR_RETENTION` (default
`device_graph=48h,network=168h,pressure=168h`). The device graph is served live,
so its ~125 KB snapshots are forensic history only; reports read at most 7 days
of network and pressure history. A window longer than the global one is ignored.

New databases use `auto_vacuum=INCREMENTAL`, and each retention pass returns
freed pages with bounded `PRAGMA incremental_vacuum(1024)` batches, taking the
write lock per batch. A database created before this mode stays in
`auto_vacuum=NONE` until one full VACUUM converts it. The owner runs that
conversion itself from the retention pass when every gate holds: free pages
are at least 1 GiB and 20% of the file, local time is inside the quiet window
(`SYSTEM_MONITOR_COMPACTION_WINDOW`, default `2-5`), and the disk holds twice
the live bytes plus the WAL plus 4 GiB. While the rewrite holds the write lock,
latest-metrics and timeline reads return `ErrMaintenance` at once, so health
answers from the in-memory snapshot instead of queueing. Each performed or
failed conversion writes `state/retention/compaction-receipt.json`.
`system-monitor maintenance compact apply --confirm` remains an optional way to
convert immediately.
