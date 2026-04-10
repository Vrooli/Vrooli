# Architecture

## Purpose

System Monitor provides real-time server monitoring with threshold-based anomaly detection, AI-driven investigation via agent-manager, and automated reporting. It collects metrics from 6 pluggable collectors, evaluates configurable thresholds, and spawns AI agents to investigate anomalies.

## Component Diagram

```
+---------------------------------------------------------+
|  UI (React + Vite)         |  CLI (Bash)                |
|  HTTP polling (5s/60s/4s)  |  system-monitor commands   |
+------------+---------------+-----------+----------------+
             |                           |
             v                           v
+---------------------------------------------------------+
|  API (Go)                                               |
|                                                         |
|  handlers/ --> services/ --> repository/                 |
|                                                         |
|  collectors/  |  agentmanager/  |  toolregistry/        |
|  middleware/  |  server/        |  config/               |
+---------+--------+--------+--------+---+----------------+
          |        |        |        |   |
          v        v        v        v   v
      Postgres  QuestDB  Redis  Ollama  agent-manager
```

## Layer Architecture

The API follows clean architecture with three main layers:

### Handlers (Presentation)
HTTP request parsing, response formatting, and route definitions. Handlers depend on narrow interfaces (`MonitorQuerier`, `InvestigationManager`, `ScriptRunner`, `ReportGenerator`, `SettingsProvider`) defined in `handlers/interfaces.go`, keeping them decoupled from concrete service types.

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
Storage abstraction via `MetricRepository`, `InvestigationRepository`, `AlertRepository`, `ReportRepository` interfaces. Currently implemented by an in-memory backend; PostgreSQL can be swapped in without changing the service layer.

`[CODE: api/internal/repository/]`

## Integration Seams

### Collector Interface
Six collectors (CPU, memory, disk, network, process, GPU) implement a common interface. New collectors are added by creating a file in `collectors/` without touching existing code. `MonitorService` accepts `WithCollectors(...)` for test injection.

`[CODE: api/internal/collectors/]`

### Agent-Manager Client
HTTP client for the agent-manager API. The `AgentExecutor` interface abstracts all agent-manager operations used by `InvestigationService`.

`[CODE: api/internal/agentmanager/]`

### Tool Registry
Pluggable tool registration (metrics_tools, investigation_tools, configuration_tools). New tools register via a registry pattern and are exposed through the tool discovery protocol.

`[CODE: api/internal/toolregistry/]`

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

- **In-memory default**: Keeps startup simple; PostgreSQL/QuestDB configured but optional
- **HTTP polling over WebSocket**: Simpler architecture, slightly higher latency (UI polls at 5s/60s/4s intervals)
- **Bash CLI with curl**: Original plan was Go CLI; actual implementation uses Bash for simplicity
- **Interface-driven services**: All service dependencies flow through interfaces for testability
- **Pluggable collectors**: New metric sources added without modifying existing code
