# swarm-manager Interoperability Audit

> Current State (2026-03-28): Active runtime interop is graph-first backlog/scenarios/settings/execution/prompts with agent-manager and optional swarm-manager. Any recommendation-endpoint references are historical context.

## Last Updated
2026-05-06

## Dependency Inventory
| Dependency | Declared | Used in Code | Required/Optional | Status |
|---|---|---|---|---|
| agent-manager | Yes (`service.json`) | `internal/agentmanager/client.go` | Required | Proto-based, discovery per-request, 20s timeout |
| swarm-manager | Yes (`service.json`) | `internal/ecosystem/client.go` | Required | JSON-based (no swarm-manager protos), discovery per-request, 20s timeout |
| knowledge-observatory | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| visited-tracker | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| scenario-completeness-scoring | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| app-issue-tracker | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| test-genie | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |
| prompt-manager | Yes (`service.json`, disabled) | Not used | Optional (P1) | N/A |

## Proto Adoption Status
- [x] All structured API request/response types use generated protos
- [x] All UI↔API communication uses fromJson/toJsonString
- [x] Protovalidate enforced at API ingress
- [x] Graph projection now uses generated proto schemas end to end
- [x] UI domain types derived from proto types (BacklogItem, Scenario, Settings, ExecutionRecord, ExecutionPolicy)

## Contract Findings
1. Non-test casts are localized to the React Flow renderer/library seam (`Record<string, unknown>` -> typed graph node data); structured UI↔API contracts do not rely on ad hoc DTO casts.
2. All UI services use `parseProtoResponse` + mapper functions through `proto-contracts.ts`.
3. All UI domain types derive from proto types via `Omit<ProtoMessage<...>, ...>` pattern.
4. Graph transport no longer uses hand-written DTOs or `Record<string, unknown>` payload maps across the UI↔API boundary.
5. Proto-contracts type guards centralize status/enum validation for all domains.
6. Agent session delete is a proto-owned REST exception:
   `DeleteAgentSessionRequest` / `DeleteAgentSessionResponse` are generated
   from `packages/proto/schemas/swarm-manager/v1/api/agent_session.proto`,
   served at `DELETE /api/v1/agent-sessions/{session_id}`, parsed in the UI via
   `deleteAgentSessionResponseSchema`, and exposed in the CLI through the
   `sessions delete --id ID --yes` command.

## UI↔API Findings
1. All hand-written interfaces in UI are component props, store state, or service interfaces — none duplicate backend graph/backlog/execution proto message shapes.
2. `fromJson` with `ignoreUnknownFields: true` in `proto-contracts.ts` accepts both proto and JSON field names.
3. All service `create`/`update` calls use `buildMessage` + `toProtoJson` for request serialization.

## Discovery/Lifecycle Findings
1. `agentmanager/client.go`: Uses `discovery.ResolveScenarioURLDefault(ctx, "agent-manager")` via `baseURLResolver` — resolved per-request, not cached at startup.
2. `ecosystem/client.go`: Uses `discovery.ResolveScenarioURLDefault(ctx, "swarm-manager")` — resolved per-request.
3. No hardcoded `localhost:port` in production integration paths.
4. Both clients use `http.NewRequestWithContext` for context propagation with 20s timeouts.
5. Dependency parity: all declared `required` dependencies in `service.json` have corresponding adapter code.

## Completed Fixes

### 2026-03-28: Graph projection proto migration and UI typing hardening
- **Proto schema** (`swarm-manager/v1/domain/graph.proto`, `swarm-manager/v1/api/graph.proto`): Added explicit graph node/edge/meta response messages and typed node payload oneofs for backlog, initiative, capture, scenario, execution, and run nodes.
- **API handler** (`internal/graph/handler.go`, `internal/graph/proto_response.go`, `internal/graph/projection.go`): Replaced raw JSON graph responses and map-based node payloads with typed projection structs encoded into proto `GraphResponse`.
- **UI graph service/store** (`ui/src/services/graph-service.ts`, `ui/src/surfaces/graph/*`): Removed hand-written graph DTOs, adopted proto schema parsing, centralized graph node typing/helpers, and added shared typed graph test builders to keep store/presentation/canvas tests aligned with the real contract.
- **Impact**: The graph-first workspace no longer drifts from the API contract and now benefits from compile-time checks across API encoding, UI mapping, clustering, presentation, and renderer tests.

### 2026-05-06: Agent session delete contract
- **Proto schema** (`swarm-manager/v1/api/agent_session.proto`): Added
  `DeleteAgentSessionRequest` and `DeleteAgentSessionResponse` for destructive
  session deletion.
- **API/UI/CLI consumers**: API handler, UI service/store, session details page,
  and CLI `sessions delete` all use the same session resource path. The UI
  parses the response through generated proto descriptors, and CLI deletion
  requires `--yes`.
- **Destructive boundary**: The operation deletes only session-owned storage and
  preserves created backlog items, initiatives, captures, files, and agent
  activity records. Active Agent Manager runs are stopped before storage
  deletion; failed stops abort deletion.

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
4. **swarm-manager JSON payloads**: `ecosystem/client.go` uses `encoding/json` with a hand-written `Task` struct because no swarm-manager proto schemas exist yet. When swarm-manager protos are created, this client should migrate to protojson.
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
