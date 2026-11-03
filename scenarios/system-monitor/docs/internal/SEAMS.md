# Seams & Architecture Boundaries

## Last Updated
2026-11-03

## Integration Seams

- **Investigation catalog seam** (`api/internal/investigations/catalog.go`):
  embeds the built-in catalog and the two shell escape hatches in the API
  binary, then overlays machine-local operator entries from storage state.
  A missing repository is not a catalog state.

- **Investigation run-history seam** (`api/internal/investigations/repository.go`):
  `Repository` owns `investigation_runs` and `investigation_findings`; the
  SQLite implementation is registered through `api/internal/modules/registry.go`.
  `Service` owns retention cutoff policy while handlers and CLI consume the
  interface.

- **Platform paging seam** (`api/internal/collectors/platform_paging_*.go`):
  exposes native cumulative paging counters, their stable consumer aliases,
  and instantaneous memory-pressure gauges (native level, paging-file usage,
  commit charge) to the pressure collector. Linux reads `/proc/vmstat`, macOS
  `host_statistics64` plus `kern.memorystatus_vm_pressure_level`, Windows PDH,
  `GlobalMemoryStatusEx`, and the low-memory notification. Pure mappings
  (`paging_*_parse.go`) are untagged so recorded fixtures for every platform
  run on any host. The collector owns counter-to-rate conversion and emits
  explicit degraded state when a backend cannot measure a counter.

- **macOS native binding** (`api/internal/darwinsys/`): cgo-free libSystem
  trampolines for `host_statistics64` and `proc_pid_rusage`; non-darwin builds
  return `ErrUnsupported`.

- **Alert evaluator seam** (`api/internal/services/alert_evaluator.go`):
  `AlertRule` implementations judge one `AlertSnapshot`; `AlertSnapshotSource`
  (satisfied by `*MonitorService`), `AlertEscalator`, `AlertRepository`, and
  `Clock` are injected, so sustain/dedup/resolve logic is tested with a stub
  clock and the in-memory repository.

- **Agent inventory seam** (`api/internal/services/orphan_agents.go`):
  `AgentInventorySource` reads the control plane's agent listing; production
  execs `vrooli agent list --json` through `CommandRunner` at most every five
  minutes. The rule reads the control plane's `orphaned` classification and
  owner and never infers ownership itself.

- **Platform fragmentation seam** (`api/internal/collectors/platform_fragmentation_*.go`):
  exposes Linux buddyinfo and compaction evidence while non-Linux builds
  return unsupported with a reason. No consumer treats absence as zero.

- **Repository Interface** (`api/internal/repository/interfaces.go`): Defines the metric-cycle, investigation, alert, report, and maintenance contracts. Memory and SQLite implement the active contracts; services do not depend on the storage engine. Testability: high — mock any repository interface. [CODE: api/internal/repository/interfaces.go]

- **Collector Interface** (`api/internal/collectors/interface.go`): All 6 collectors (CPU, memory, disk, network, process, GPU) implement a common interface. New collectors can be added without touching existing code. `MonitorService` accepts `WithCollectors(...)` option to inject test doubles, skipping real OS collectors. Testability: high — mock collectors for unit tests. Disk collectors are observational only: they may expose pressure and attribution signals, but broad cleanup/remediation belongs to storage-manager policy and audit, not system-monitor mutation paths. [CODE: api/internal/collectors/interface.go]

- **Agent-Manager Client** (`api/internal/agentmanager/client.go`): HTTP client for agent-manager API. `AgentExecutor` interface (`api/internal/services/interfaces.go`) abstracts all agent-manager operations used by InvestigationService. Testability: high — mock `AgentExecutor` for unit tests. [CODE: api/internal/agentmanager/client.go]

- **MetricsSource** (`api/internal/services/interfaces.go`): `MetricsSource` interface provides on-demand metrics to InvestigationService via `GetCurrentMetricsFresh()`. `*MonitorService` satisfies it. Testability: high — mock for unit tests. [CODE: api/internal/services/interfaces.go]

- **Alert Notification** (`api/internal/services/alert.go`): `sendWebhook()` and `sendNotifications()` are the notification dispatch points. Email channel is referenced but not implemented. `HTTPDoer` interface (`api/internal/services/interfaces.go`) makes the HTTP client injectable. Testability: high — mock `HTTPDoer` for webhook tests. [CODE: api/internal/services/alert.go]

- **Infrastructure Provider** (`api/internal/infrastructure/provider.go`): Injected into `MonitorService` via `infrastructure.Provider` interface. Collects database pool, HTTP pool, message queue, and storage I/O metrics. Testability: high — mock the `Provider` interface. [CODE: api/internal/infrastructure/provider.go]

- **Clock** (`api/internal/services/clock.go`): `Clock` interface (`Now()`, `Since()`) abstracts time operations. `RealClock` for production, `StubClock` for deterministic testing with `Advance(d)` and `Set(t)` methods. Injected into all five services (MonitorService, InvestigationService, AlertService, ReportService, SettingsManager) via functional options (`WithMonitorClock`, `WithInvestigationClock`, etc.). Testability: high — eliminates flaky time-dependent tests and enables deterministic cooldown/ID testing. [CODE: api/internal/services/clock.go]

- **Environment** (`api/internal/config/config.go`): the `Environment.LookupEnv` seam owns process environment access. `LoadWithEnvironment` builds the validated startup config, which retains the injected reader for runtime values. `systemEnvironment` is the production adapter; map-backed readers exercise runtime overrides in tests. Runtime, handlers, service clients, and collectors consume the resulting config rather than reading process globals. [CODE: api/internal/config/config.go]
- **Logger** (`api/internal/logx/logger.go`): API components depend on the small structured `logx.Logger` interface. The server composition root creates the configured `slog.Logger` and injects component-scoped loggers into handlers, middleware, repositories, and services. `logx.Default` is the compatibility adapter for startup paths before composition; tests can inject a recording logger. [CODE: api/internal/logx/logger.go]
- **CLI watch clock** (`cli/domains/overview/register.go`): `handlers.Now` supplies the wall clock rendered by `system-monitor watch`. Production wires `time.Now`; command tests can provide a fixed instant. [CODE: cli/domains/overview/register.go]
- **ConfigStore** (`api/internal/services/configstore.go`): `ConfigStore` abstracts durable configuration file I/O. `FileConfigStore` is the production adapter; `MemoryConfigStore` is used by settings and prompt-store tests. Investigation trigger configuration no longer uses this store; investigation prompt loading remains injectable through `WithPromptStore`. [CODE: api/internal/services/configstore.go]

- **MaintenanceRepository** (`api/internal/repository/maintenance.go`): Narrow interface (`EstimateMetricRetention`, `PruneMetricsBefore`, `SQLiteStats`, `Compact`) for the metrics storage lifecycle, embedded in the aggregate `Repository`. SQLite implements all four (prune in a tx, `VACUUM` under the write mutex); the in-memory backend implements estimate/prune and returns `ErrNotSupported` for stats/compaction. `MetricsMaintenanceService` (`api/internal/services/maintenance.go`) owns cutoff computation (via `Clock`) and the destructive-operation confirmation contract on top of it. Testability: high — drive the service over the in-memory or sqlite repo. [CODE: api/internal/repository/maintenance.go]

- **RetentionScheduler** (`api/internal/services/retention_scheduler.go`): Owns *when* scheduled retention runs (startup + live `retention_check_interval_seconds`), delegating storage work to `MetricsMaintenanceService`. Replaces the former repository-owned cleanup goroutine, so timing decisions live in the service layer. Settings are read live each cycle. Testability: high — inject settings + a maintenance service over a fake repo. [CODE: api/internal/services/retention_scheduler.go]

- **CommandRunner** (`api/internal/services/command.go`): `CommandRunner` interface abstracts OS command execution (`Run(ctx, name, args, dir) → stdout, stderr, exitCode, err`). `ExecCommandRunner` wraps `os/exec` for production. Injected into `ScriptService` via optional constructor parameter. Testability: high — mock `CommandRunner` to test script execution without running real bash. [CODE: api/internal/services/command.go]

## Responsibility Zones

- **Entry/presentation**: `api/internal/handlers/` — HTTP request parsing, response formatting. `api/internal/middleware/` for CORS, logging, auth.
- **Coordination/orchestration**: `api/internal/services/` — MonitorService, InvestigationService, ReportService, AlertService, SettingsService coordinate business logic.
- **Domain rules**: `api/internal/models/` — Data structures, threshold definitions, investigation state machine.
- **Integrations/infrastructure**: `api/internal/agentmanager/` (external agent-manager API), `api/internal/repository/` (storage), `api/internal/collectors/` (OS-level metric collection), `api/internal/infrastructure/` (infrastructure monitoring).
- **Cross-cutting concerns**: `api/internal/config/` (configuration), `api/internal/server/` (router, middleware wiring, shutdown, database, logging). Storage and prompt paths are centralized in `api/internal/services/paths.go` (`ResolveRuntimeStateBasePath`, `ResolvePromptBasePath`).

## Decision Points

- **Anomaly threshold evaluation**: In `api/internal/services/monitor.go`. Compares collected metrics against configurable notification rules. Thresholds and per-rule investigation policy are owned by System Settings; delivery is owned by Notification Hub. The old trigger CRUD API returns migration guidance and is not a settings store.
- **Investigation cooldown**: In `api/internal/services/investigation.go`. Prevents investigation flooding based on configurable period. Exposed via cooldown endpoints. Test coverage: `TestCooldownEnforced`, `TestCooldownReset`, `TestCooldownStatus` in `investigation_test.go`; `TestAlertCooldownPreventsSpam`, `TestAlertCooldownExpires`, `TestCleanupCooldowns` in `alert_test.go`.
- **Storage backend selection**: In `api/internal/server/runtime.go`. SQLite is the durable default. Non-production tests may explicitly select memory with `SYSTEM_MONITOR_STORAGE_MODE=memory`; production rejects that mode. The greenfield schema is declarative and idempotent. Test coverage: repository cycle and fresh-schema tests.
- **Script execution**: In `api/internal/services/script.go`. Executes bash scripts with timeout. Test coverage: `TestExecuteScript_Success`, `TestExecuteScript_Failure`, `TestExecuteScript_Timeout` in `script_test.go`.

## Change Axes

- **Primary change axis**: New metric collectors, new investigation scripts, threshold tuning.
- **Current cost of change**: Well-localized for new collectors (add file in `collectors/`). Investigation scripts are filesystem-based (add to `investigations/active/`). API endpoint changes require touching `handlers/` + `server/router.go`.

## Observability Surface

- **Logging**: Structured logging via `api/internal/server/logging.go` middleware. Logs request/response.
- **Health checks**: `/health` and `/api/v1/health` endpoints.
- **Signal gaps**: No OpenTelemetry/distributed tracing. No formal event bus for investigation lifecycle events. Monitor self-metrics now expose cycle duration, forks, skipped/failed/stale work, persistence latency, and headroom.

## Architecture Clarity Notes

- Clean architecture with handler → service → repository layers.
- Service-layer dependencies flow through interfaces (`MetricsSource`, `AgentExecutor`, `HTTPDoer`, `infrastructure.Provider`, `repository.*`, `Clock`, `ConfigStore`, `CommandRunner`, `config.Environment`), enabling isolated unit testing. Environment-sensitive runtime configuration is resolved once through `config.LoadWithEnvironment`.
- Handler layer uses narrow interfaces (`MonitorQuerier`, `InvestigationManager`, `ScriptRunner`, `ReportGenerator`, `SettingsProvider`) defined in `api/internal/handlers/interfaces.go`, decoupling handlers from concrete service types.
- SQLite is the durable default; explicit development memory mode is available for isolated tests.
- Agent-manager integration is well-isolated in its own package.
- UI uses HTTP polling (not WebSocket) — simpler but higher latency.

## Exploration Log

- Refactored from monolithic main.go to modular internal/ packages.
- Original plan was Go CLI; actual implementation is Bash CLI with curl.
- Persistent time-series storage is implemented through SQLite cycle-linked observations with bounded maintenance paths.

## Target-aware scenario reads

Metrics and device-graph reads use the same typed protobuf request/response
shape for the local service and a remote target. Remote requests go through
`api-core/nodereach.Client.CallScenario`, which owns Bridge discovery,
authentication, target binding, catalog admission, timeout, and response
limits. The handler does not relay a CLI command or parse stdout; `Dispatch`
and `Open` remain separate because they are job and interactive-session verbs.

## UI and API contract seam

Structured System Monitor UI/API traffic uses generated Connect contracts,
`protoFetch`, and the parsers in `ui/src/shared/api/proto-contracts.ts`.
Health probes, development profiling, raw forensics, and log browsing remain
intentional REST exceptions because they are operational or stream-like
surfaces rather than scenario-domain entities. The UI/API boundary should not
reintroduce hand-written domain DTOs or unvalidated casts.

## UI lifecycle and shared utility seams

Polling, health checks, metric history, and investigation-agent state are split
into focused hooks; shared formatters, timestamp/chart helpers, API converters,
health/map utilities, and modal/polling primitives own cross-surface behavior.
Lifecycle cleanup and race prevention belong in those hooks and their focused
tests, not in individual dashboard components.

Typed REST exceptions use parser functions at the shared `apiFetch` boundary:
health, forensics, log browsing, and machine inventory validate untrusted JSON
before exposing feature types. Router navigation state is parsed by the metrics
feature before use. `apiFetch` without a parser deliberately returns `unknown`;
callers that need domain types must provide a parser. Abort errors use the
shared `isAbortError` helper so cancellation checks do not drift between hooks.

## Monitor collection seams

The monitor service owns collection-cycle orchestration. Process sampling and
retention of independently ranked CPU, memory, paging, pinned-owner, and GPU
leaders live in `api/internal/services/monitor_processsampling.go`. Metric
history, freshness, and pressure projections live in
`api/internal/services/monitor_metricqueries.go`. Linux per-core baseline,
counter-reset, and core-set behavior lives in `api/internal/collectors/platform_cpu_linux.go`
so those observation rules can be tested without reading host procfs files.

## Persistence projection seam

`api/internal/repository/metric_projection.go` owns the API projection from a
collector observation to `MetricsResponse` and `MetricState`. SQLite and memory
repositories delegate to this projector so their storage encoding cannot drift
into different field coverage or observation semantics. SQLite schema/setup
stays in `repository.go`; cycle/history queries and record persistence are
separate in `metrics.go` and `records.go`. The memory adapter separates cycle
projection from record persistence in `repository.go` and `records.go`. The
cross-backend contract test lives at `api/internal/repository/repository_conformance_test.go`.

## Background service lifecycle

`api/internal/services/service_loop.go` owns the shared stopped → running →
stopping → stopped lifecycle for restartable background loops. Alert evaluation
and scheduled retention provide only their loop bodies; the helper enforces one
active generation, cancellation, join receipts, and safe restart after exit.
Services with additional owned work, such as threshold recovery waiters, retain
their specialized shutdown coordination around their own lifecycle.

`ReportService` passes the request context through time-range discovery,
historical reads, and report persistence. Daily and weekly windows share one
start-time selector; unavailable history falls back to the requested window,
while cancellation stops report generation.
