# Seams & Architecture Boundaries

## Last Updated
2026-02-16

## Integration Seams

- **Repository Interface** (`api/internal/repository/interfaces.go`): Defines `MetricRepository`, `InvestigationRepository`, `AlertRepository`, `ReportRepository`. Currently only `memory/repository.go` implements them. PostgreSQL implementation can be swapped in without changing service layer. Testability: high — mock any repository interface.

- **Collector Interface** (`api/internal/collectors/interface.go`): All 6 collectors (CPU, memory, disk, network, process, GPU) implement a common interface. New collectors can be added without touching existing code. Testability: high — mock collectors for unit tests.

- **Agent-Manager Client** (`api/internal/agentmanager/client.go`): HTTP client for agent-manager API. Can be replaced with a mock client for testing investigation flows without a running agent-manager. Testability: medium — requires interface extraction for clean mocking.

- **Alert Notification** (`api/internal/services/alert.go`): `sendWebhook()` and `sendNotifications()` are the notification dispatch points. Email channel is referenced but not implemented. Testability: medium — webhook URL can be pointed at a test server.

- **Infrastructure Provider** (`api/internal/infrastructure/provider.go`): Collects database pool, HTTP pool, message queue, and storage I/O metrics. Can be stubbed for testing. Testability: medium.

- **Tool Registry** (`api/internal/toolregistry/`): Pluggable tool registration (metrics_tools, investigation_tools, configuration_tools). New tools register via registry pattern. Testability: high — tools are independent units.

## Responsibility Zones

- **Entry/presentation**: `api/internal/handlers/` — HTTP request parsing, response formatting. `api/internal/toolhandlers/` for tool discovery. `api/internal/middleware/` for CORS, logging, auth.
- **Coordination/orchestration**: `api/internal/services/` — MonitorService, InvestigationService, ReportService, AlertService, SettingsService coordinate business logic.
- **Domain rules**: `api/internal/models/` — Data structures, threshold definitions, investigation state machine.
- **Integrations/infrastructure**: `api/internal/agentmanager/` (external agent-manager API), `api/internal/repository/` (storage), `api/internal/collectors/` (OS-level metric collection), `api/internal/infrastructure/` (infrastructure monitoring).
- **Cross-cutting concerns**: `api/internal/config/` (configuration), `api/internal/server/` (router, middleware wiring, shutdown, database, logging).

## Decision Points

- **Anomaly threshold evaluation**: In `api/internal/services/monitor.go`. Compares collected metrics against configurable thresholds. Triggers are managed via `/api/v1/investigations/triggers` endpoints. Test coverage: no unit tests exist.
- **Investigation cooldown**: In `api/internal/services/investigation.go`. Prevents investigation flooding based on configurable period. Exposed via cooldown endpoints. Test coverage: no unit tests exist.
- **Storage backend selection**: In `api/internal/server/runtime.go`. Defaults to in-memory repository. PostgreSQL can be configured via DATABASE_URL. Test coverage: no unit tests exist.

## Change Axes

- **Primary change axis**: New metric collectors, new investigation scripts, threshold tuning.
- **Current cost of change**: Well-localized for new collectors (add file in `collectors/`). Investigation scripts are filesystem-based (add to `investigations/active/`). API endpoint changes require touching `handlers/` + `server/router.go`.

## Observability Surface

- **Logging**: Structured logging via `api/internal/server/logging.go` middleware. Logs request/response.
- **Health checks**: `/health` and `/api/v1/health` endpoints.
- **Signal gaps**: No OpenTelemetry/distributed tracing. No structured error codes. No metrics about the monitor itself (meta-monitoring). No formal event bus for investigation lifecycle events.

## Architecture Clarity Notes

- Clean architecture with handler → service → repository layers.
- In-memory default keeps startup simple but means data is lost on restart.
- Agent-manager integration is well-isolated in its own package.
- Tool discovery protocol adds agent-composability surface.
- UI uses HTTP polling (not WebSocket) — simpler but higher latency.

## Exploration Log

- Refactored from monolithic main.go to modular internal/ packages (documented in api/REFACTORING.md).
- Original plan was Go CLI; actual implementation is Bash CLI with curl.
- QuestDB was chosen over InfluxDB for performance; actual usage defaults to in-memory.
