# swarm-manager Interoperability Audit

> Current State (2026-02-14): Active runtime interop is backlog/scenarios/settings/execution with agent-manager and optional ecosystem-manager. Any recommendation-endpoint references are historical context.

## Last Updated
2026-02-13

## Dependency Inventory
| Dependency | Declared | Used in Code | Required/Optional | Status |
|---|---|---|---|---|
| agent-manager | Yes (`service.json`) | `internal/agentmanager/client.go` | Required | Proto-based, discovery per-request, 20s timeout |
| ecosystem-manager | Yes (`service.json`) | `internal/ecosystem/client.go` | Required | JSON-based (no ecosystem-manager protos), discovery per-request, 20s timeout |
| knowledge-observatory | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| visited-tracker | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| scenario-completeness-scoring | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| app-issue-tracker | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| test-genie | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| prompt-manager | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |

## Proto Adoption Status
- [x] All API request/response types use generated protos
- [x] All UI↔API communication uses fromJson/toJsonString
- [x] Protovalidate enforced at API ingress
- [x] No unsafe type assertions
- [x] UI domain types derived from proto types (BacklogItem, Scenario, Settings, ExecutionRecord, ExecutionPolicy)

## Contract Findings
1. No unsafe casts (`as any`, `as SomeType`) in non-test code (API or UI).
2. All UI services use `parseProtoResponse` + mapper functions through `proto-contracts.ts`.
3. All UI domain types derive from proto types via `Omit<ProtoMessage<...>, ...>` pattern.
4. Proto-contracts type guards centralize status/enum validation for all domains.

## UI↔API Findings
1. All hand-written interfaces in UI are component props, store state, or service interfaces — none duplicate proto message shapes.
2. `fromJson` with `ignoreUnknownFields: true` in `proto-contracts.ts` accepts both proto and JSON field names.
3. All service `create`/`update` calls use `buildMessage` + `toProtoJson` for request serialization.

## Discovery/Lifecycle Findings
1. `agentmanager/client.go`: Uses `discovery.ResolveScenarioURLDefault(ctx, "agent-manager")` via `baseURLResolver` — resolved per-request, not cached at startup.
2. `ecosystem/client.go`: Uses `discovery.ResolveScenarioURLDefault(ctx, "ecosystem-manager")` — resolved per-request.
3. No hardcoded `localhost:port` in production integration paths.
4. Both clients use `http.NewRequestWithContext` for context propagation with 20s timeouts.
5. Dependency parity: all declared `required` dependencies in `service.json` have corresponding adapter code.

## Completed Fixes

### 2026-02-13: Execution service backlogItem data loss fix
- **Root cause**: `execution/service.go` defined its own `backlogItem` struct (line 525) missing the `created` and `research_target` fields. When `updateBacklogStatus()` wrote this incomplete struct back to `spec.json` via `storage.WriteJSONAtomic()`, those fields were silently dropped. Any item that went through the execution pipeline (queue, cancel, complete, fail) permanently lost its `created` timestamp and research target.
- **Impact**: Items with blank `created` fail the `BacklogItem.created` protovalidate constraint (`min_len = 1`) on the frontend, causing `parseProtoResponse` to reject them. The backlog list appeared empty (`{}`) even when items existed on disk.
- **Fix**: Added `Created string json:"created"` and `ResearchTarget string json:"research_target,omitempty"` to the `backlogItem` struct in `execution/service.go` to match the canonical field set in `backlog/handler.go`. Now round-tripping through `loadBacklogItem` → `updateBacklogStatus` preserves all fields.
- **Note (2026-03-26)**: The `research_target` field was subsequently removed from the data model as part of the research backlog item rework. The `updateBacklogStatus()` function now deletes `research_target` from on-disk data to clean up legacy values.

### 2026-02-13: Backlog queue proto migration
- **Proto schema** (`backlog.proto`): Added `mode`, `delay_seconds`, `started_by` fields to `QueueBacklogItemRequest` — previously only `operation` was declared while the Go handler expected all four fields via a hand-written struct.
- **API handler** (`internal/backlog/handler.go`): Replaced hand-written `queueBacklogRequest` struct + `json.NewDecoder` with `httputil.DecodeProtoJSON` + `httputil.ValidateProtoRequest` using the generated `QueueBacklogItemRequest`. Protovalidate now enforces field constraints at ingress.
- **UI backlog-service.ts**: `queue()` now uses `buildMessage(QueueBacklogItemRequestSchema, ...)` + `toProtoJson()` instead of raw JSON object.

### 2026-02-13: Execution policy proto serialization
- **UI execution-policy-service.ts**: `update()` now uses `buildMessage(ExecutionPolicySchema, ...)` + `toProtoJson()` instead of raw `{ default_mode, default_delay_seconds }` object.

### 2026-02-13: Execution module proto migration
- **API handler** (`internal/execution/handler.go`): Replaced `json.NewDecoder`/`httputil.JSON` with `httputil.DecodeProtoJSON`/`httputil.ProtoJSON` for all execution endpoints. Added `recordToProto()` and `policyToProto()` converters. Create endpoint now uses `httputil.ValidateProtoRequest` for ingress validation.
- **UI execution-service.ts**: Removed hand-written DTOs (`ExecutionRecordDTO`, `ExecutionListResponse`, `ExecutionItemResponse`). Now uses `parseProtoResponse` + `mapProtoExecutionRecord` for all response parsing. Create request serialized via proto `buildMessage` + `toProtoJson`.
- **UI execution-policy-service.ts**: Removed hand-written `ExecutionPolicyResponse` DTO. Now uses `parseProtoResponse` + `mapProtoExecutionPolicy`.
- **UI types/domain.ts**: `ExecutionRecord` and `ExecutionPolicy` now derived from proto types (like `BacklogItem` pattern) instead of hand-written interfaces.
- **UI proto-contracts.ts**: Added execution schemas (`listExecutionResponseSchema`, `executionResponseSchema`, `executionPolicyResponseSchema`) and mapper functions (`mapProtoExecutionRecord`, `mapProtoExecutionPolicy`) with type guards.

### 2026-02-13: Ecosystem client hardening
- **ecosystem/client.go**: Replaced `HTTPDoer.Post(url, contentType, body)` with `HTTPDoer.Do(req)` pattern (consistent with agent-manager client). Added `http.NewRequestWithContext` for context propagation. Default client uses `&http.Client{Timeout: 20 * time.Second}`.

### 2026-02-13: Agent-manager dead code removal
- Removed legacy JSON types: `ResearchRequest`, `ResearchResponse`, `createTaskRequest`, `taskPayload`, `createTaskResponse`, `createRunRequest`, `createRunResponse`.
- Removed legacy methods: `CreateResearchRun`, `createTask`, `createRun`.
- Removed `Client` interface (only proto-based `HTTPClient` methods remain: `CreateTask`, `CreateRun`, `GetRun`, `StopRun`, `Health`, `EnsureProfile`, `ResolveURL`).
- Updated tests to cover proto-based methods only.

### 2026-02-05: Initial proto adoption
- Recommendations, settings, backlog, scenarios, and agent-manager status endpoints migrated to proto contracts with protovalidate at ingress.
- UI domain types derived from generated proto types to reduce drift risk.
- UI proto parsing accepts both proto field names and JSON field names via `fromJson` mapping.

## Remaining Items
1. **String-to-enum migration**: `scenario.proto`, `backlog.proto`, and `execution.proto` encode lifecycle states as strings with `in:` constraints; full proto enums would be safer but require a deprecation/migration plan.
2. **Agent-manager `UseProtoNames: false`**: The agent-manager client uses `lowerCamelCase` JSON field names (not proto snake_case) because agent-manager expects camelCase. This is intentional but should be documented/tested to prevent accidental changes.
3. **File content endpoints**: File read/write endpoints intentionally use raw/streamed responses rather than proto wrappers.
4. **Ecosystem-manager JSON payloads**: `ecosystem/client.go` uses `encoding/json` with a hand-written `Task` struct because no ecosystem-manager proto schemas exist yet. When ecosystem-manager protos are created, this client should migrate to protojson.
5. **Go inter-scenario retry policy**: Neither `agentmanager` nor `ecosystem` clients implement retry/backoff (single attempt, then error). For required dependencies this fail-fast behavior is acceptable, but bounded retry on transient/transport errors would improve resilience.

## Proper/Complete Gates
- [x] Contract safety (all boundaries use proto types + protovalidate)
- [x] Discovery/addressing safety (per-request resolution, no hardcoded ports)
- [x] Dependency parity (code ↔ service.json aligned)
- [x] Envelope/status normalization (centralized type guards + mappers)
- [x] UI↔API contract safety (generated types, fromJson with ignoreUnknownFields, no hand-written duplicates)
- [ ] Runtime recovery tests (Go clients lack retry/backoff tests; UI has `fetchWithRetry`)

## Notes
- All inter-service HTTP clients now use context-aware requests with explicit timeouts.
- Proto field name convention: swarm-manager API uses `UseProtoNames: true` (snake_case); agent-manager client uses `UseProtoNames: false` (camelCase) to match agent-manager's API.
- UI `store-utils.ts` provides `fetchWithRetry` with exponential backoff for all store-based data fetching.
