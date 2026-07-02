# Seams & Architecture Boundaries

## Last Updated
2026-06-02

## Integration Seams

- **Repository Interface** (`api/internal/repository/interfaces.go`): Defines `MetricRepository`, `InvestigationRepository`, `AlertRepository`, `ReportRepository`. Currently only `memory/repository.go` implements them. PostgreSQL implementation can be swapped in without changing service layer. Testability: high — mock any repository interface. [CODE: api/internal/repository/interfaces.go]

- **Collector Interface** (`api/internal/collectors/interface.go`): All 6 collectors (CPU, memory, disk, network, process, GPU) implement a common interface. New collectors can be added without touching existing code. `MonitorService` accepts `WithCollectors(...)` option to inject test doubles, skipping real OS collectors. Testability: high — mock collectors for unit tests. [CODE: api/internal/collectors/interface.go]

- **Agent-Manager Client** (`api/internal/agentmanager/client.go`): HTTP client for agent-manager API. `AgentExecutor` interface (`api/internal/services/interfaces.go`) abstracts all agent-manager operations used by InvestigationService. Testability: high — mock `AgentExecutor` for unit tests. [CODE: api/internal/agentmanager/client.go]

- **MetricsSource** (`api/internal/services/interfaces.go`): `MetricsSource` interface provides on-demand metrics to InvestigationService via `GetCurrentMetricsFresh()`. `*MonitorService` satisfies it. Testability: high — mock for unit tests. [CODE: api/internal/services/interfaces.go]

- **Alert Notification** (`api/internal/services/alert.go`): `sendWebhook()` and `sendNotifications()` are the notification dispatch points. Email channel is referenced but not implemented. `HTTPDoer` interface (`api/internal/services/interfaces.go`) makes the HTTP client injectable. Testability: high — mock `HTTPDoer` for webhook tests. [CODE: api/internal/services/alert.go]

- **Infrastructure Provider** (`api/internal/infrastructure/provider.go`): Injected into `MonitorService` via `infrastructure.Provider` interface. Collects database pool, HTTP pool, message queue, and storage I/O metrics. Testability: high — mock the `Provider` interface. [CODE: api/internal/infrastructure/provider.go]

- **Clock** (`api/internal/services/clock.go`): `Clock` interface (`Now()`, `Since()`) abstracts time operations. `RealClock` for production, `StubClock` for deterministic testing with `Advance(d)` and `Set(t)` methods. Injected into all five services (MonitorService, InvestigationService, AlertService, ReportService, SettingsManager) via functional options (`WithMonitorClock`, `WithInvestigationClock`, etc.). Testability: high — eliminates flaky time-dependent tests and enables deterministic cooldown/ID testing. [CODE: api/internal/services/clock.go]

- **ConfigStore** (`api/internal/services/configstore.go`): `ConfigStore` interface abstracts configuration file I/O (read/write). `FileConfigStore` reads/writes from disk; `MemoryConfigStore` provides an in-memory implementation for tests. Injected into `InvestigationService` (via `WithConfigStore`) and `SettingsManager` (via `WithSettingsConfigStore`). A separate `promptStore` field on `InvestigationService` (via `WithPromptStore`) abstracts prompt template loading. Testability: high — inject `MemoryConfigStore` to test config read/write without filesystem. [CODE: api/internal/services/configstore.go]

- **MaintenanceRepository** (`api/internal/repository/maintenance.go`): Narrow interface (`EstimateMetricRetention`, `PruneMetricsBefore`, `SQLiteStats`, `Compact`) for the metrics storage lifecycle, embedded in the aggregate `Repository`. SQLite implements all four (prune in a tx, `VACUUM` under the write mutex); the in-memory backend implements estimate/prune and returns `ErrNotSupported` for stats/compaction. `MetricsMaintenanceService` (`api/internal/services/maintenance.go`) owns cutoff computation (via `Clock`) and the destructive-operation confirmation contract on top of it. Testability: high — drive the service over the in-memory or sqlite repo. [CODE: api/internal/repository/maintenance.go]

- **RetentionScheduler** (`api/internal/services/retention_scheduler.go`): Owns *when* scheduled retention runs (startup + live `retention_check_interval_seconds`), delegating storage work to `MetricsMaintenanceService`. Replaces the former repository-owned cleanup goroutine, so timing decisions live in the service layer. Settings are read live each cycle. Testability: high — inject settings + a maintenance service over a fake repo. [CODE: api/internal/services/retention_scheduler.go]

- **CommandRunner** (`api/internal/services/command.go`): `CommandRunner` interface abstracts OS command execution (`Run(ctx, name, args, dir) → stdout, stderr, exitCode, err`). `ExecCommandRunner` wraps `os/exec` for production. Injected into `ScriptService` via optional constructor parameter. Testability: high — mock `CommandRunner` to test script execution without running real bash. [CODE: api/internal/services/command.go]

## Responsibility Zones

- **Entry/presentation**: `api/internal/handlers/` — HTTP request parsing, response formatting. `api/internal/middleware/` for CORS, logging, auth.
- **Coordination/orchestration**: `api/internal/services/` — MonitorService, InvestigationService, ReportService, AlertService, SettingsService coordinate business logic.
- **Domain rules**: `api/internal/models/` — Data structures, threshold definitions, investigation state machine.
- **Integrations/infrastructure**: `api/internal/agentmanager/` (external agent-manager API), `api/internal/repository/` (storage), `api/internal/collectors/` (OS-level metric collection), `api/internal/infrastructure/` (infrastructure monitoring).
- **Cross-cutting concerns**: `api/internal/config/` (configuration), `api/internal/server/` (router, middleware wiring, shutdown, database, logging). Path resolution centralized in `api/internal/services/paths.go` (`ResolveConfigBasePath`, `ResolvePromptBasePath`, `ResolveScriptsDir`).

## Decision Points

- **Anomaly threshold evaluation**: In `api/internal/services/monitor.go`. Compares collected metrics against configurable thresholds. Triggers are managed via `/api/v1/investigations/triggers` endpoints. Test coverage: trigger CRUD tested in `investigation_test.go`.
- **Investigation cooldown**: In `api/internal/services/investigation.go`. Prevents investigation flooding based on configurable period. Exposed via cooldown endpoints. Test coverage: `TestCooldownEnforced`, `TestCooldownReset`, `TestCooldownStatus` in `investigation_test.go`; `TestAlertCooldownPreventsSpam`, `TestAlertCooldownExpires`, `TestCleanupCooldowns` in `alert_test.go`.
- **Storage backend selection**: In `api/internal/server/runtime.go`. Defaults to in-memory repository. PostgreSQL can be configured via DATABASE_URL. Test coverage: no unit tests exist.
- **Script execution**: In `api/internal/services/script.go`. Executes bash scripts with timeout. Test coverage: `TestExecuteScript_Success`, `TestExecuteScript_Failure`, `TestExecuteScript_Timeout` in `script_test.go`.

## Change Axes

- **Primary change axis**: New metric collectors, new investigation scripts, threshold tuning.
- **Current cost of change**: Well-localized for new collectors (add file in `collectors/`). Investigation scripts are filesystem-based (add to `investigations/active/`). API endpoint changes require touching `handlers/` + `server/router.go`.

## Observability Surface

- **Logging**: Structured logging via `api/internal/server/logging.go` middleware. Logs request/response.
- **Health checks**: `/health` and `/api/v1/health` endpoints.
- **Signal gaps**: No OpenTelemetry/distributed tracing. No structured error codes. No metrics about the monitor itself (meta-monitoring). No formal event bus for investigation lifecycle events.

## Architecture Clarity Notes

- Clean architecture with handler → service → repository layers.
- All service-layer dependencies flow through interfaces (`MetricsSource`, `AgentExecutor`, `HTTPDoer`, `infrastructure.Provider`, `repository.*`, `Clock`, `ConfigStore`, `CommandRunner`), enabling isolated unit testing.
- Handler layer uses narrow interfaces (`MonitorQuerier`, `InvestigationManager`, `ScriptRunner`, `ReportGenerator`, `SettingsProvider`) defined in `api/internal/handlers/interfaces.go`, decoupling handlers from concrete service types.
- In-memory default keeps startup simple but means data is lost on restart.
- Agent-manager integration is well-isolated in its own package.
- UI uses HTTP polling (not WebSocket) — simpler but higher latency.

## Exploration Log

- Refactored from monolithic main.go to modular internal/ packages.
- Original plan was Go CLI; actual implementation is Bash CLI with curl.
- QuestDB was chosen over InfluxDB for performance; actual usage defaults to in-memory.
