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
Rates come from the pressure collector's shared counter-rate tracker, so the
first sample after start is `not_yet_sampled` on every platform. "Sustained"
is the alert evaluator's job, not the collector's.

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
`threshold_check_interval`, evaluates every rule against the scheduler-owned
snapshot, and persists `condition` alerts. The CLI (`status`, `alerts`,
`watch`) and the dashboard alert panel read `AlertsService.ListAlerts`; no
client derives an alert from a metric value.

| Rule | Condition |
|------|-----------|
| `metrics_snapshot` | critical when no snapshot exists or it is older than ten collection intervals |
| `cpu_usage` | warning ≥ `cpu_threshold`, critical ≥ `cpu_threshold` + 10 |
| `memory_usage` | warning ≥ `memory_threshold`, critical ≥ `memory_threshold` + 5 |
| `tcp_connections` | warning ≥ 5,000, critical ≥ 20,000, and warning on a leak-shaped climb over the last 10 minutes (≥ 5 samples, ends ≥ 500 and ≥ 2× its start, within 10% of its peak, no single drop larger than half the rise). Hundreds of connections are normal on this host, so there is no low fixed level. |
| `gpu_usage` | warning ≥ 90% (unsupported without a GPU reading) |
| `memory_pressure` | elevated → warning, critical → critical, unknown → rule unsupported |
| `orphaned_agents` | reads `vrooli agent list --json` (`state`, `kernel_state`, `owner{kind,ref,…}`, `memory_bytes`, `swap_bytes`, `reapable`, `reap_reason`). Warning when any `state=orphaned` row is older than 1 h; critical when orphaned rows hold ≥ 2 GiB of known memory+swap or a `reapable=true` row has stayed reapable for 1 h (auto-reap off or failing). Null or absent memory is reported unknown, never 0. Unsupported ("control plane does not classify orphans yet") when rows carry no `owner`. |

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
runtime scanner find it. It never spawns an agent-manager investigation;
that remains an operator action. System Monitor never kills processes.

**Events.** Disk/CPU band transitions from the threshold scheduler and
escalation anomalies are point-in-time records (`threshold_violation`,
`anomaly`) created resolved. They keep history and notifications without
leaving rows that never close; `ListAlerts` includes them only with
`include_events`. Rows written before this change remain open in storage and
are excluded from the default view by type.

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
        |
        v
Threshold Evaluation (warning/critical levels)
        |
        +-- below threshold --> store metrics, continue polling
        |
        +-- above threshold --> AlertEvaluator (sustain, dedup, resolve)
                                    |   -> AlertService (persist, notify)
                                    |
                                    v
                        InvestigationService.Trigger()
                                    |
                                    v
                        AgentExecutor.SpawnAgent() --> agent-manager
                                    |
                                    v
                        Investigation runs (steps, findings, progress)
                                    |
                                    v
                        Results stored in repository
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
