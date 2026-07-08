# Interoperability Audit: system-monitor

**Date**: 2026-02-17 (updated 2026-06-25, bright-window closure)
**Scenario**: system-monitor
**Dependencies**: agent-manager (required)

## Communication Paths

| Path | Protocol | Status |
|------|----------|--------|
| API → agent-manager | HTTP + protojson | Good |
| UI → API | Connect JSON for proto-owned calls + REST exceptions | Good |
| CLI → API | Generated Connect clients + manifest bindings | Good |
| API public contract | Generated Connect handlers on `http.ServeMux` + explicit REST exceptions | Good |

## Findings

### Bright-window final route/proto state (2026-06-25)

The proto layer is authored and generated for 8 services, and every proto-owned
domain is mounted through generated Connect handlers:

- `HealthService`
- `MetricsService`
- `ReportsService`
- `SettingsService`
- `CapacityService`
- `MaintenanceService`
- `InvestigationsService`
- `ScriptsService`

The API router now uses the standard library `http.ServeMux`; gorilla/mux is no
longer a dependency. Manual REST routes for proto-owned metrics, reports,
settings, capacity, maintenance, investigations, and scripts were removed. A
live smoke confirmed `GET /api/v1/metrics/current` returns `404`, while
`/vrooli.system_monitor.v1.metrics.MetricsService/GetCurrentMetrics` succeeds.

#### Resolved in bright-window

- **Metrics process timeline was code-ahead-of-proto**: `GetProcessTimeline`
  was added to `MetricsService`, generated, implemented, mounted, bound in the
  CLI manifest, and migrated to the generated CLI client.
- **Cooldown mutations and agent stop were route-ahead-of-proto**:
  `ResetCooldown`, `UpdateCooldownPeriod`, and `StopAgent` now have explicit
  `InvestigationsService` RPCs.
- **CLI raw calls were removed**: `metrics`, `reports`, `settings`, `capacity`,
  `maintenance`, `investigations`, and `overview` use generated Connect clients.
- **UI proto-owned calls no longer rely on legacy REST routes**:
  `protoFetch()` maps existing hook paths to Connect JSON procedure paths while
  preserving raw REST exceptions for logs/forensics/tools.

#### Remaining intentional REST exceptions

| Surface | Runtime path | Reason |
|---|---|---|
| Health probes | `/health`, `/api/v1/health` | Ops/lifecycle probes should remain simple HTTP GETs. |
| Development profiling | `/debug/pprof/*` | Dev-only diagnostics, disabled in production. |
| Forensics | `/api/v1/forensics/*` | Raw host diagnostics, not a typed scenario-domain contract. |
| Logs | `/api/v1/logs*` | Raw log browsing/stream-like shapes, not covered by current proto services. |

#### Remaining non-blocking drift

| Drift | Current state | Decision |
|---|---|---|
| Process kill | UI references `POST /processes/{pid}/kill`, but no API route exists. | Keep as follow-up safety design work. |

### Resolved

#### F0: Disk detail Connect method implemented (LOW)
- **Handler** (`api/internal/handlers/metrics.go`): `MetricsService.GetDiskDetail` now returns read-only partition/detail data.
- **Boundary**: Response notes point remediation to cleanup-manager plan/apply; system-monitor still observes only and does not mutate disk state.

#### F1: Hardcoded `localhost:port` in investigation prompt and handler response (HIGH)
- **Handler** (`api/internal/handlers/investigations.go`): `resolveAPIBaseURL()` derives URL from forwarded headers (resolved 2026-02-17).
- **Service** (`api/internal/services/investigation.go`): `resolveAPIBaseURL()` now reads `config.Server.APIBaseURL`, loaded from `API_BASE_URL` env var with fallback to `http://localhost:{API_PORT}`. Eliminates hardcoded localhost in investigation prompts.
- **Config** (`api/internal/config/config.go`): `ServerConfig.APIBaseURL` added.

#### F2: No proto schemas for system-monitor's own domain (MEDIUM)
- **Created**: `packages/proto/schemas/system-monitor/v1/` with 6 proto files:
  - `domain/types.proto` — enums (InvestigationStatus, Severity, FindingType, etc.)
  - `domain/investigations.proto` — Investigation, InvestigationStep, TriggerConfig, etc.
  - `domain/metrics.proto` — MetricsResponse, DetailedMetrics, CPUMetrics, etc.
  - `api/investigations.proto` — request/response wrappers
  - `api/metrics.proto` — request/response wrappers
  - `api/service.proto` — gRPC service definition with HTTP annotations
- **Generated**: Go, TypeScript, and Python bindings via `buf generate`.
- **UI wired**: `ui/src/types/api.ts` re-exports proto-generated types as the single source of truth.
- **Go status constants**: `models.StatusQueued`, `models.StatusCompleted`, etc. replace scattered string literals. `models.IsTerminalStatus()` centralizes terminal-status checks.

#### F4: Investigation status values stringly-typed across UI and API (MEDIUM)
- **UI File**: `ui/src/features/investigations/hooks/useInvestigationAgents.ts`
- **Types File**: `ui/src/types/api.ts`
- Terminal statuses were inline with extra values (`"canceled"`) not sent by the API.
- **Fix**: `INVESTIGATION_TERMINAL_STATUSES` in `types/api.ts` with aligned values. Go side uses `models.IsTerminalStatus()` — the `"canceled"` variant is eliminated.

#### F7: Status vocabulary drift — raw string literals instead of constants (MEDIUM)
- Multiple Go files used raw status string literals instead of `models.Status*` constants, including a **bug** where `"pending"` was used as a status value even though it's not a defined status (correct value: `"queued"`).
- **Files fixed**:
  - `services/investigation.go`: `"pending"` → `models.StatusQueued` (bug fix)
  - `services/report.go`: `"completed"` → `models.StatusCompleted`

#### F8: Go handlers use encoding/json, not protojson (HIGH → RESOLVED)
- All handler files used `respondWithJSON`/`respondWithError` (defined in `health.go`) or raw `json.NewEncoder(w).Encode()` and `http.Error()`.
- **Fix**: Created `api/internal/httputil/` package with:
  - `response.go` — `JSON()`, `JSONWithStatus()`, `Error()`, `BadRequest()`, `NotFound()`, `InternalError()`, `Conflict()`, `ServiceUnavailable()`
  - `protojson.go` — `ProtoJSON()`, `ProtoJSONWithStatus()`, `DecodeProtoJSON()`, `ValidateProto()`, `ValidateProtoRequest()`
- All 5 handler files migrated to use httputil: `health.go`, `investigations.go`, `metrics.go`, `reports.go`, `settings.go`.
- Old `respondWithJSON`/`respondWithError` functions removed.
- Proto-aware `ProtoJSON()` uses `protojson.MarshalOptions{UseProtoNames: true}` for canonical snake_case output.

#### F9: No proto schemas for reports, scripts, settings domains (MEDIUM → RESOLVED)
- **Created**: 3 new domain proto files + 3 new API proto files:
  - `domain/scripts.proto` — `InvestigationScript`, `ScriptExecution`, `ScriptExecutionStatus`
  - `domain/reports.proto` — `EnhancedSystemReport`, `ReportTimeRange`, `PerformanceAnalysis`, `MetricStats`, `Trend`, etc.
  - `domain/settings.proto` — `SystemSettings`
  - `api/scripts.proto` — Request/response wrappers
  - `api/reports.proto` — Request/response wrappers
  - `api/settings.proto` — Request/response wrappers (including maintenance state)
- Updated `api/service.proto` with all new RPCs.
- Generated Go, TypeScript, Python bindings.

#### F10: UI uses raw fetch + type casts without schema validation (MEDIUM → RESOLVED)
- **Created**: `ui/src/shared/api/proto-contracts.ts` — Parse helpers using `fromJson()` with `ignoreUnknownFields: true` for all domain types.
- **Created**: `protoFetch()` in `ui/src/shared/api/apiFetch.ts` — Proto-aware fetch variant that deserializes via parse helpers.
- **Updated**: `ui/src/types/api.ts` — All hand-written interfaces replaced with proto re-exports. Script, report, and settings types now proto-backed.

#### F3: UI agent payload parsing overly defensive (MEDIUM → RESOLVED)
- `mapAgentPayload()` and `parseAgentsResponse()` replaced with clean `toAgentState()` and `extractAgents()` in `useInvestigationAgents.ts`.
- All camelCase fallbacks removed — reads snake_case fields directly matching the Go API contract.
- `extractBoolean`/`extractNumber`/`extractString` type guard imports removed (no longer needed).

#### F11: UI hooks use raw apiFetch instead of proto-validated protoFetch (MEDIUM → RESOLVED)
- `useSystemMonitor.ts`: Generic `handleApiCall<T>()` replaced with direct `protoFetch()` calls for 5 endpoints (metrics, detailed metrics, processes, infrastructure, investigations). Field accesses updated to camelCase matching proto types.
- `useMetricHistory.ts`: `apiFetch<MetricsTimelineResponse>` replaced with `protoFetch + parseMetricsTimelineResponse`. Proto `Timestamp` objects converted to ISO strings via `timestampDate()`.
- `DiskDetailView.tsx`: `apiFetch<DiskDetailResponse>` replaced with `protoFetch + parseDiskDetailResponse`. All partition, watcher, and usage entry field accesses updated to camelCase.
- `proto-contracts.ts`: Added `parseMetricsTimelineResponse` and `parseDiskDetailResponse` parse helpers.

#### F12: Go handlers emit encoding/json, not protojson for domain objects (HIGH → RESOLVED)
- All handler files migrated from `httputil.JSON()` to `httputil.ProtoJSON()` for proto-typed domain responses.
- Created `api/internal/convert/` package with `InvestigationToProto()`, `SettingsToProto()`, `MetricsResponseToProto()`, etc. for converting internal Go structs to proto message types.
- Wrapper responses (settings, maintenance state) that include non-proto fields (`success`, `error`) remain on `httputil.JSON()`.

#### F13: Settings/maintenance handlers use encoding/json + hand-written response structs (HIGH → RESOLVED)
- `handlers/settings.go`: Removed `SettingsResponse`, `MaintenanceStateRequest`, `MaintenanceStateResponse` structs.
- All 5 settings/maintenance handlers now use proto-generated API types (`apipb.GetSettingsResponse`, `apipb.SetMaintenanceStateResponse`, etc.) + `httputil.ProtoJSON()`/`httputil.ProtoJSONWithStatus()`.
- Request decoding uses `httputil.DecodeProtoJSON()` instead of `json.NewDecoder().Decode()`.
- Added `convert.ProtoToSettings()` reverse conversion.

#### F14: TriggerInvestigation returns map[string]interface{} (MEDIUM → RESOLVED)
- `handlers/investigations.go`: `TriggerInvestigation` response replaced with `apipb.TriggerInvestigationResponse` + `httputil.ProtoJSONWithStatus()`.

#### F15: UI settings/triggers/maintenance/agent hooks use hand-written types (MEDIUM → RESOLVED)
- `SystemSettingsModal.tsx`: Removed `SystemSettings` and `SettingsResponse` interfaces; replaced with proto-typed `create(SystemSettingsSchema)` + `protoFetch` + API response parse helpers.
- `AutomaticTriggersSection.tsx`: Removed `TriggerApiResponse` and `CooldownApiResponse`; reads now use `protoFetch` + `parseGetTriggersResponse`/`parseGetCooldownStatusResponse`.
- `useSystemMonitor.ts`: Removed `MaintenanceStateResponse`; maintenance GET/POST use `protoFetch` + `parseGetMaintenanceStateResponse`/`parseSetMaintenanceStateResponse`.
- `useHealthCheck.ts`: Maintenance toggle uses `protoFetch` + `parseSetMaintenanceStateResponse`.
- `useInvestigationAgents.ts`: Agent reads parse responses through `parseInvestigation` with typed `protoToAgentState` mapper.
- `InvestigationsPanel.tsx`: Trigger POST uses `apiFetch` instead of raw `fetch`.
- `InvestigationScriptsPanel.tsx`: Scripts reads use `apiFetch` instead of raw `fetch`.
- `proto-contracts.ts`: Added 9 API-level parse helpers for settings, maintenance, triggers, cooldown, and investigations wrapper types.

#### F16: GetCooldownStatus returns bare CooldownStatus instead of GetCooldownStatusResponse wrapper (HIGH — BUG)
- **Handler** (`api/internal/handlers/investigations.go:GetCooldownStatus`): Returned bare `convert.CooldownStatusToProto(status)` instead of wrapping in `apipb.GetCooldownStatusResponse{Cooldown: ...}`.
- **Impact**: UI parsed response as `GetCooldownStatusResponse` and read `.cooldown` field, which was always empty. Cooldown data silently failed to load.
- **Fix**: Handler now wraps the proto in `&apipb.GetCooldownStatusResponse{Cooldown: convert.CooldownStatusToProto(status)}`.

#### F17: Metrics handlers still use httputil.JSON() despite proto schemas existing (MEDIUM → RESOLVED)
- `GetDetailedMetrics`, `GetProcessMonitor`, `GetInfrastructureMonitor` had TODO comments saying "migrate to ProtoJSON when proto schemas cover these types" — but schemas already existed.
- **Fix**: Created comprehensive converter functions in `convert.go`:
  - `DetailedMetricsToProto()` — with nested CPU, Memory, Network, GPU, SystemHealth converters
  - `ProcessMonitorDataToProto()` — with ProcessHealthInfo converter
  - `InfrastructureMonitorDataToProto()` — with MessageQueueInfo, StorageIOInfo converters
  - Plus 20+ helper converters for all nested types (ProcessInfo, TCPConnectionStates, ConnectionPool, GPUDeviceMetrics, etc.)
- All 3 handlers now use `httputil.ProtoJSON()` + `convert.*ToProto()`.

#### F18: Raw fetch() calls in UI hooks and components (MEDIUM → RESOLVED)
- **useInvestigationAgents.ts**: 4 raw `fetch()` calls replaced:
  - `GET /investigations/agent/current` → `protoFetch` + `parseInvestigation`
  - `POST /investigations/agent/spawn` → `protoFetch` + `parseTriggerInvestigationResponse`
  - `POST /investigations/agent/{id}/stop` → `protoFetch` + `InvestigationsService.StopAgent`
  - `GET /investigations/agent/{id}/status` → `protoFetch` + `parseInvestigation`
- **Fragile multi-shape response parsing eliminated**: `extractAgents()` tried 4+ response shapes (array, `{agents:[]}`, `{agent:{}}`, bare object with `id`/`investigation_id`). Replaced with direct `protoFetch` + `protoToAgentState()` — single code path, proto-validated.
- **AutomaticTriggersSection.tsx**: 5 raw `fetch()` calls replaced with `apiFetch`:
  - Toggle trigger, toggle auto-fix, update cooldown period, update threshold, reset cooldown.
- **ProcessMonitor.tsx**: 1 raw `fetch()` for process kill replaced with `apiFetch`.
- **useScriptExecution.ts**: 1 raw `fetch()` for script execution replaced with `apiFetch`.
- **Only remaining raw fetch**: `useSystemMonitor.ts` cleanup effect (fire-and-forget on unmount — appropriate for this pattern).

#### F19: Status string duplication in useInvestigationsSectionState.ts (LOW → RESOLVED)
- Inline hardcoded array `['completed', 'error', 'failed', 'stopped', 'cancelled', 'canceled']` replaced with `INVESTIGATION_TERMINAL_STATUSES` set from `types/api.ts`.
- Removed invalid values: `'error'` (not a valid API status) and `'canceled'` (misspelling — API uses `'cancelled'`).

#### F20: Scripts endpoints return placeholder JSON, not proto responses (HIGH → RESOLVED)
- `ListScripts`, `GetScript`, `ExecuteScript` handlers returned `map[string]interface{}` or 404 stubs.
- **Fix**: Created `services/script.go` (`ScriptService`) that discovers scripts from `investigations/active/`, parses metadata from script headers, and executes scripts with timeout.
- Added `convert.ScriptMetaToProto()`, `convert.ScriptExecutionToProto()` converters.
- All 3 handlers now return proto responses (`ListScriptsResponse`, `GetScriptResponse`, `ExecuteScriptResponse`) via `httputil.ProtoJSON()`.
- Proto schema updated: `GetScriptResponse` now includes `content` field; `ExecuteScriptRequest` now includes optional `content` field for overrides.

#### F21: 10 Go handlers decode request bodies with json.NewDecoder instead of proto (MEDIUM → RESOLVED)
- Migrated to `httputil.DecodeProtoJSON()` with proto request types:
  - `TriggerInvestigation` → `TriggerInvestigationRequest`
  - `UpdateInvestigationStatus` → `UpdateInvestigationStatusRequest` (with enum→string conversion)
  - `UpdateInvestigationFindings` → `UpdateInvestigationFindingsRequest` (with Struct→map conversion)
  - `UpdateInvestigationProgress` → `UpdateInvestigationProgressRequest`
  - `AddInvestigationStep` → `AddInvestigationStepRequest` (with proto→model step conversion)
  - `UpdateTrigger` → `UpdateTriggerRequest`
  - `ExecuteScript` → `ExecuteScriptRequest`
  - `GenerateReport` → `GenerateReportRequest`
- Handlers without matching proto types kept as-is: `UpdateCooldownPeriod`, `UpdateTriggerThreshold`, `UpdateAgentConfig`.

#### F22: Response envelope inconsistency — map[string]string instead of proto types (MEDIUM → RESOLVED)
- Replaced `httputil.JSON(w, map[string]string{"status": "..."})` with proto response types:
  - `UpdateInvestigationStatus` → `UpdateInvestigationStatusResponse`
  - `UpdateInvestigationFindings` → `UpdateInvestigationFindingsResponse`
  - `UpdateInvestigationProgress` → `UpdateInvestigationProgressResponse`
  - `AddInvestigationStep` → `AddInvestigationStepResponse`
  - `UpdateTrigger` → `UpdateTriggerResponse`
- Endpoints without proto response types kept as-is: `UpdateTriggerThreshold`.

#### F23: UI scripts hooks use apiFetch with unsafe casts instead of protoFetch (MEDIUM → RESOLVED)
- `InvestigationScriptsPage.tsx`: `apiFetch<{ scripts?: InvestigationScript[] }>` replaced with `protoFetch + parseListScriptsResponse`. Script content fetch replaced with `protoFetch + parseGetScriptResponse`.
- `InvestigationScriptsPanel.tsx`: Same pattern for `loadScripts` and `openScript`.
- `useScriptExecution.ts`: `apiFetch<Record<string, unknown>>` replaced with `protoFetch + parseExecuteScriptResponse`. Removed all manual field extraction (`extractString`, `extractNumber`) and `as ScriptExecution` unsafe casts.

#### F24: Local TriggerConfig interface shadows proto TriggerConfig type (LOW → RESOLVED)
- `AutomaticTriggersSection.tsx`: Renamed local `interface TriggerConfig` to `TriggerCardConfig` to avoid confusion with proto-exported `TriggerConfig` from `types/api.ts`.

### Documented (no code change)

#### F5: Resource URLs use hardcoded localhost defaults (LOW)
- `api/internal/config/config.go:141-146` — resource URLs (Postgres, Redis, QuestDB) default to `localhost:<port>`.
- These are **resources** (not scenarios), so discovery doesn't apply. Defaults are correct for local dev; production overrides via env vars.

#### F6: Health endpoint self-reference uses hardcoded localhost (LOW)
- `api/internal/config/config.go:298` — `loadHealthEndpoints()` builds `http://localhost:<API_PORT>/health`.
- Self-check within the same process; localhost is appropriate.

## What's Good (no changes needed)
- **Agent-manager client** (`api/internal/agentmanager/client.go`): Proto-generated types, `protojson` marshal/unmarshal with `DiscardUnknown: true`, per-request `discovery.ResolveScenarioURLDefault()`.
- **Terminal status handling**: Uses proto enum constants (`RUN_STATUS_COMPLETE`, etc.) — no stringly-typed comparisons.
- **Dependency declaration**: `agent-manager` properly declared as required in `.vrooli/service.json`.
- **No hardcoded ports** for inter-scenario calls.
- **Agent-manager `UseProtoNames: false`**: Intentional — matches agent-manager's expected lowerCamelCase convention.
- **httputil package**: Centralized JSON/ProtoJSON response helpers (mirrors swarm-manager pattern).
- **UI proto-contracts**: Parse helpers with `fromJson()` + `protoFetch()` for schema-validated API consumption.
- **Full protojson coverage**: All Go handlers serving domain objects now use `httputil.ProtoJSON()` with `UseProtoNames: true`.
- **No raw fetch() in UI feature code**: All API calls go through `apiFetch` or `protoFetch` (except fire-and-forget cleanup).

## Completion Gates
- [x] No hardcoded `localhost` in non-config, non-test API responses
- [x] Terminal statuses centralized and aligned with API contract
- [x] Proto schemas for system-monitor domain types (investigations + metrics)
- [x] Proto schemas for reports, scripts, settings domains
- [x] Status string literals replaced with `models.Status*` constants
- [x] Go handlers migrated to httputil centralized response helpers
- [x] UI proto-contracts layer with parse helpers and `protoFetch()`
- [x] UI types re-exported from proto-generated types (single source of truth)
- [x] Migrate individual handler responses from `httputil.JSON()` to `httputil.ProtoJSON()` (domain object responses)
- [x] Replace UI `apiFetch<T>()` calls with `protoFetch()` + parse helpers in hooks
- [x] Remove defensive camelCase fallbacks in `mapAgentPayload()`
- [x] Migrate settings/maintenance wrapper responses to proto API types + ProtoJSON
- [x] Wire `protoFetch`/`apiFetch` for settings, triggers, maintenance, and agent endpoints
- [x] Fix GetCooldownStatus contract mismatch (bare vs wrapper response)
- [x] Migrate GetDetailedMetrics/GetProcessMonitor/GetInfrastructureMonitor to ProtoJSON
- [x] Replace all raw fetch() in UI with apiFetch/protoFetch
- [x] Deduplicate status string comparisons using INVESTIGATION_TERMINAL_STATUSES
- [x] Wire `protoFetch` for scripts endpoints
- [x] Migrate request decoding to proto types (`DecodeProtoJSON`)
- [x] Standardize confirmation responses with proto response types
- [x] Rename local TriggerConfig → TriggerCardConfig to avoid proto shadow
- [ ] Health endpoint proto schema (operational/diagnostic, not a domain contract — deferred)
