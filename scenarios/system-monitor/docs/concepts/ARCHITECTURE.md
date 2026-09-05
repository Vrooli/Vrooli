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
        +-- above threshold --> AlertService (notifications, cooldown)
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
