# ecosystem-manager Interoperability Audit

## Last Updated
2026-02-13 (second pass)

## Dependency Inventory
| Dependency | Declared | Used in Code | Required/Optional | Status |
|---|---|---|---|---|
| agent-manager | Yes (service.json) | Yes (pkg/agentmanager/client.go) | Required | OK - proto-based serialization, per-request discovery |
| prompt-manager | Yes (service.json) | Yes (pkg/autosteer/prompt_loader.go) | Optional | OK - graceful degradation with fallback instructions |
| visited-tracker | Yes (service.json) | Yes (pkg/handlers/visited_tracker.go) | Optional | OK - proxy pattern with discovery |
| postgres | Yes (resource) | Yes (pkg/server/server.go, database/sql) | Required | OK - resource dependency |
| qdrant | Yes (resource) | Yes (heuristic) | Required | OK - resource dependency |
| claude-code | Yes (resource) | Yes (queue processor) | Required | OK - resource dependency |
| ollama | Yes (resource) | Yes (heuristic) | Required | OK - resource dependency |
| openrouter | Yes (resource) | Yes (heuristic) | Required | OK - resource dependency |
| minio | Yes (resource) | Yes (heuristic) | Required | OK - resource dependency |

## Contract Findings

### L0 Schema Contract
1. **Agent-manager client** (`pkg/agentmanager/client.go`): Uses `protojson` for serialization with proto-generated types. `DiscardUnknown: true` for forward compatibility. Good.
2. **Proto schemas defined** for ecosystem-manager domain and API types in `packages/proto/schemas/ecosystem-manager/v1/`. 8 domain schemas (task, settings, queue, autosteer, insights, discovery, prompt, common) and 8 API schemas (tasks, settings, queue, autosteer, insights, discovery, prompts, executions). Generated types available in Go, TypeScript, and Python. Good.
3. **UI proto-contracts** (`ui/src/lib/proto-contracts.ts`): Uses `@bufbuild/protobuf` `fromJson()` with `@bufbuild/protovalidate` for response validation at the API boundary. Falls back to manual normalization when proto validation fails (e.g., runtime-enriched responses with extra fields). Good.

### L1 Serialization Contract
1. `pkg/agentmanager/client.go`: Uses `protojson.MarshalOptions{UseProtoNames: false}` (lowerCamelCase) to match agent-manager HTTP handler expectations, and `protojson.UnmarshalOptions{DiscardUnknown: true}` for forward-compatible deserialization. Note: protojson unmarshal accepts both snake_case and camelCase regardless of options, so the asymmetry is safe.
2. All other inter-scenario communication uses standard `encoding/json`. Acceptable — no proto contracts exist for visited-tracker or prompt-manager HTTP APIs.

### L2 Envelope & Status Semantics
1. **Agent-manager client**: Uses typed proto status enums (`domainpb.RunStatus_RUN_STATUS_COMPLETE`, `_FAILED`, `_CANCELLED`) for terminal state detection in `WaitForRun` and `buildExecuteResult`. No stringly-typed status comparisons. Good.
2. **buildExecuteResult** (`service.go`): Centralized status-to-result mapping. Error heuristics (rate_limited, timeout, max_turns) use case-insensitive substring matching on error messages. Tested in `service_test.go`. Good.
3. Visited-tracker proxy passes responses through unchanged — no status normalization needed. Good.

### L3 Discovery & Addressing
1. **Agent-manager client**: Uses `discovery.ResolveScenarioURLDefault` per-request in `resolveBaseURL()`. Good.
2. **Visited-tracker handlers**: Uses `discovery.ResolveScenarioURLDefault` per-request for API proxy, `discovery.ResolveScenarioURL` with `UI_PORT` key for UI port endpoint. Good.
3. **Prompt-loader**: Uses per-request URL resolution via `resolvePromptManagerURL()` which calls `discovery.ResolveScenarioURLDefault`. Env var `PROMPT_MANAGER_URL` override for testing. Good.
4. No hardcoded `localhost:port` in production integration paths.

### L4 Lifecycle Dependency Contract
1. All three scenario dependencies declared in `service.json` match code usage. Good.
2. `visited-tracker` declared as optional — proxy handlers return 503 when unavailable. Good.
3. `prompt-manager` declared as optional — loader falls back to hardcoded instructions. Good.
4. **Fixed**: Removed `scenario-completeness-scoring` forward declaration — no API code referenced it.

### L5 Runtime Recovery
1. **Agent-manager client**: Re-resolves URL per-request via discovery. Good.
2. **Prompt-loader**: `syncAll()` re-resolves URL per-request. `loadPrompt()` retries sync after 30s cooldown when unavailable. Good.
3. **Visited-tracker handlers**: Re-resolves URL per-request via discovery. Shared HTTP client with 30s timeout. Good.

## UI↔API Findings

### Hardcoded URLs
- No hardcoded `localhost:port` in UI source. API base URL resolved via `resolveWithConfig()` from `@vrooli/api-base`. Good.

### Type Safety
- Proto schemas defined in `packages/proto/schemas/ecosystem-manager/v1/` generate types for Go, TypeScript, and Python. UI view-model interfaces in `ui/src/types/api.ts` remain as the consumer-facing types (snake_case fields used by 66+ component files).
- UI fetch layer (`ui/src/lib/api.ts`) uses proto-contracts (`ui/src/lib/proto-contracts.ts`) for response validation via `@bufbuild/protovalidate`. All API boundary calls parse responses through centralized proto-contract functions: `parseTaskResponse`, `parseExecutionResponse`, `parseSettingsResponse`, `parseQueueStatusResponse`, `parseRunningProcessResponse`, `parseResourceResponse`, `parseScenarioResponse`, and `parseActiveTargetResponse`. Proto validation failures fall back to manual field mapping.
- All fetch calls use standard `fetch()` → `res.json()` pattern via `ApiClient.fetchJSON()`. Low risk for intra-scenario boundary.

### Casing
- API returns snake_case consistently via Go `json:"field_name"` struct tags. UI proto-contracts use `fromJson()` with `ignoreUnknownFields: true` which accepts snake_case JSON and maps to camelCase proto fields. Mapping functions then convert back to snake_case UI types. Fallback normalizers handle both casings defensively.

## Discovery/Lifecycle Findings
1. Verified via `rg "localhost:[0-9]+"` — no hardcoded scenario ports in integration code paths. Startup logs use `0.0.0.0` placeholder (cosmetic only).
2. All scenario clients use `api-core/discovery` for URL resolution.

## Fixes Applied (Current Session — Interop Steer Pass 2)
1. **Eliminated unsafe `as Resource`/`as Scenario` casts** (L1/Medium): `getResources()` and `getScenarios()` in `ui/src/lib/api.ts` previously used inline `as Resource`/`as Scenario` casts with `any`-typed items, bypassing proto validation. Replaced with centralized `parseResourceResponse()` and `parseScenarioResponse()` in proto-contracts, which attempt `fromJson(ResourceSchema, ...)` / `fromJson(ScenarioSchema, ...)` with fallback normalization.
2. **Eliminated unsafe `as any` casts in `getActiveTargets()`** (L1/Medium): Replaced manual `(entry as any).target` field extraction with centralized `parseActiveTargetResponse()` in proto-contracts, using proper type guards (`isTaskStatus`).
3. **Fixed `getSystemInsights()` return type** (L1/Low): Changed return type from `Promise<any>` to `Promise<SystemInsightReport>`, eliminating an untyped API boundary.
4. **Fixed UI API base URL caching** (L3/Low): Replaced async promise-cache `getApiBase()` + `ensureApiBase()` pattern with synchronous `resolveApiBase()` from `@vrooli/api-base`, matching all other scenarios. Eliminates the anomalous permanent cache and asymmetry with the WebSocket layer.
5. **Migrated execution handlers to proto response types** (L0/Low): 4 handlers in `tasks_execution.go` now return proto-generated types (`ExecutionHistoryListResponse`, `ExecutionPromptResponse`, `ExecutionOutputResponse`) instead of ad-hoc `map[string]any`. New `proto_convert.go` provides `executionHistoryToProto()` for `queue.ExecutionHistory` → `domain.ExecutionRecord` conversion. JSON output keys unchanged (snake_case via proto `json` tags).

## Fixes Applied (Proto Schema Session)
1. **Added proto schemas for ecosystem-manager** (L0/Critical): Created 16 proto schema files (8 domain, 8 API) in `packages/proto/schemas/ecosystem-manager/v1/`. Generated types for Go, TypeScript, JavaScript, and Python (64 files). Closes the L0 gap where ecosystem-manager had no shared type contract.
2. **Added UI proto-contracts layer** (L0/High): Created `ui/src/lib/proto-contracts.ts` with `@bufbuild/protovalidate` validation, mapping functions (proto camelCase → UI snake_case), and fallback normalization. Replaces ~235 lines of defensive normalization code in `ui/src/lib/api.ts`.
3. **Migrated API client to proto-based parsing** (L1/High): All `ApiClient` methods in `ui/src/lib/api.ts` now use proto-contracts for response parsing (`parseTaskResponse`, `parseExecutionResponse`, `parseSettingsResponse`, etc.) instead of hand-written normalization functions.

## Fixes Applied (Prior Interop Audit Session)
1. **Removed unused dependency declaration** (L4/Medium): Removed `scenario-completeness-scoring` from `service.json` — no API code referenced it. Eliminates a dependency parity defect.
2. **Added buildExecuteResult tests** (L2/Medium): New `service_test.go` with 16 test cases covering terminal status classification (COMPLETE/FAILED/CANCELLED/RUNNING/PENDING), error heuristics (rate_limited, timeout, max_turns), summary extraction, and duration calculation. Previously untested envelope/status normalization seam.
3. **Fixed INTEROP_AUDIT factual error** (L1/Low): Previous audit incorrectly claimed `UseProtoNames: true` — actual code uses `UseProtoNames: false` (lowerCamelCase to match agent-manager). Corrected with rationale.

## Fixes Applied (Prior Session)
1. **Dependency declaration parity** (L4/Critical): Added `visited-tracker` as optional dependency in `service.json`. Was called in code but not declared.
2. **Hardcoded localhost in UI port response** (L3/Medium): `GetVisitedTrackerUIPortHandler` now uses `discovery.ResolveScenarioURL` instead of `fmt.Sprintf("http://localhost:%d", port)`.
3. **HTTP client reuse** (L5/Medium): `VisitedTrackerHandlers` now uses a shared `http.Client` with 30s timeout instead of creating a new client per-request.
4. **Bare http.Get without timeout** (L5/Medium): `GetCampaignsForTargetHandler` now uses the shared client (with timeout) instead of `http.Get()`.
5. **Prompt-loader per-request URL resolution** (L3/Medium): `syncAll()` and `ReadSkillsWithScope()` now re-resolve the prompt-manager URL via discovery on each call.
6. **Startup log hardcoded localhost** (L3/Low): Changed cosmetic startup log messages from `localhost` to `0.0.0.0`.

## Remaining Risks
1. **Go handlers not yet fully using proto response types** (L0/Low): Execution history handlers now use proto types. ~30 remaining handlers (tasks CRUD, settings, queue, autosteer, discovery, insights) still use `map[string]any` or embed Go structs directly. Proto types exist — migration is incremental and low risk since UI proto-contracts handle current shapes via fallback normalization.
2. **Visited-tracker types not in proto** (L0/Accepted): Campaign/TrackedFile types are proxied transparently from visited-tracker via `io.Copy` — ecosystem-manager never inspects or transforms the response bytes. These types belong to visited-tracker's own contract. Adding proto schemas for types owned by another service would create a cross-ownership problem. **Closed as accepted architectural decision.**

## Proper/Complete Gates
- [x] Contract safety - proto schemas defined for all domain and API types; UI uses proto-contracts with validation
- [x] Discovery/addressing safety - no hardcoded scenario ports; all scenario clients resolve per-request
- [x] Dependency parity - all 3 scenario deps declared in service.json match code usage
- [x] Envelope/status normalization - centralized in `buildExecuteResult` with typed enum comparison; tested
- [x] UI↔API contract safety - no hardcoded URLs; proto-generated types with runtime validation at API boundary
- [x] Runtime recovery - per-request URL resolution for all scenario adapters; shared HTTP clients with timeouts
