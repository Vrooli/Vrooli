# agent-inbox Interoperability Audit

## Last Updated
2026-02-13 (re-verified against current code)

## Dependency Inventory
| Dependency | Declared | Used in Code | Required/Optional | Status |
|---|---|---|---|---|
| agent-manager | Yes (service.json) | Yes (integrations/agent_manager.go, handlers/agent_mode.go) | Optional | OK - graceful degradation |
| scenario-to-cloud | Yes (service.json) | No direct protocol client; command discovery flows through Search Hub | Optional | OK - graceful degradation |
| prompt-manager | Yes (service.json) | Yes (services/prompt_sync.go, config/config.go) | Optional | OK - graceful degradation |
| ollama | Yes (resource) | Yes (integrations/ollama.go, services/skill_suggest.go) | Required | OK - resource dependency |
| openrouter | Yes (resource) | Yes (integrations/openrouter.go) | Required | OK - resource dependency |

## Contract Findings

### L0 Schema Contract
1. The old provider Tool Discovery Protocol proto types have been removed. Agent Inbox now injects Search Hub command discovery results as completion context instead of loading provider tool manifests.
2. `AgentManagerClient` uses generated proto types (`apipb.CreateTaskRequest`, `apipb.CreateRunResponse`, `apipb.GetRunResponse`, `apipb.GetRunEventsResponse`, `domainpb.ContinueRunRequest`) with `protojson` marshal/unmarshal for all agent-manager communication. Compile-time type safety for all request/response shapes.

### L1 Serialization Contract
1. `agent_manager.go:28-31`: Module-level `protoMarshalOpts` and `protoUnmarshalOpts` with `UseProtoNames: true` and `DiscardUnknown: true`. Good.
2. `openrouter.go:283-284`: `ConvertMessages` uses `m["role"].(string)` type assertions on untyped message maps. **Risk: low** - internal-only path, but panics on unexpected input.

### L2 Envelope & Status Semantics
1. `reconciliation.go` uses `integrations.ProtoRunStatusToLocal()` and `integrations.IsActiveRunStatus()` from `agent_manager.go` — centralized, proto-driven status mapping.
2. `GetRunStatus()` uses proto `RunStatus` enum via `ProtoRunStatusToLocal()` — no string normalization needed.
3. `TranslateProtoEvent` uses typed proto oneof accessors instead of `map[string]interface{}` with camelCase keys. Event type mapping uses proto `RunEventType` enum.
4. `async_tracker.go:processStatusResult`: Uses configurable runtime `CompletionConditions` for terminal status detection. Good - config-driven, not hardcoded.

### L3 Discovery & Addressing
1. `AgentManagerClient` re-resolves URL on connection failure via `reResolveURL()` + `getBaseURL()`. Good.
2. `PromptSyncService.Sync()` re-resolves prompt-manager URL via `api-core/discovery` on connection failure. Good.
3. `PromptSyncService` CRUD methods (`CreateSkillInPromptManager`, `UpdateSkillInPromptManager`, `DeleteSkillInPromptManager`, `RecordUsage`) attempt URL re-resolution when URL is empty. Good.
4. `config/config.go:getPromptManagerURL`: Uses `api-core/discovery.ResolveScenarioURLDefault`. Good.

### L4 Lifecycle Dependency Contract
1. All three scenario dependencies (agent-manager, scenario-to-cloud, prompt-manager) are declared in `service.json` with `required: false`. Code matches - all three degrade gracefully when unavailable. Good.
2. Search Hub command discovery degrades with diagnostics when Search Hub is unavailable. Good.

### L5 Runtime Recovery
1. `AgentManagerClient` re-resolves URL on connection errors (connection refused, DNS failure, dial errors). Good.
2. `StartAgentChat` run-creation HTTP call re-resolves on connection failure. Good.
3. `PromptSyncService` re-resolves via `api-core/discovery` on sync failure across all code paths (Sync, CRUD, RecordUsage). Good.
4. `resilience/` package: Provides retry, circuit breaker, and fallback patterns. Good infrastructure, not yet wired into adapters.
5. `ReconciliationService`: Creates fresh `AgentManagerClient` per reconciliation cycle. Good.
6. `AsyncTrackerService`: Recovers operations from database on startup, does fresh status check before resuming polling. Good.

## UI↔API Findings

### Hand-Written Interfaces
The old provider tool manifest interfaces and generated proto equivalents were
removed with the Tool Discovery Protocol cleanup. Remaining UI interfaces model
agent-inbox-owned REST/SSE shapes or OpenRouter request/response shapes.

### Interfaces That Are Legitimately Hand-Written (No Proto Equivalent)
These types are internal to agent-inbox (SQLite domain) or represent API response shapes constructed by agent-inbox's own handlers:
- `Chat`, `Message`, `Label`, `ToolCallRecord`, `Attachment`, `ChatWithMessages` — internal domain
- `StreamingEvent` — SSE protocol definition
- `ToolCall` — OpenAI function calling format
- `AgentChatConfig`, `AgentModeResponse`, `AgentModeStatus`, `AgentEvent`, `AgentEventsResponse` — agent-inbox translates agent-manager proto types into these before sending to UI
- `Model`, `ModelPricing`, `ModelArchitecture` — OpenRouter model metadata
- `Template`, `Skill`, `UsageStats` and related — internal features

### UI Serialization
- All fetch calls use raw `fetch()` → `res.json()` with no `fromJson` / `useProtoNames` — acceptable because:
  1. The Go API serializes with `protojson.MarshalOptions{UseProtoNames: true}`, producing snake_case JSON
  2. The UI hand-written interfaces expect snake_case fields (matching the API output)
  3. No casing mismatch exists in practice

### UI Unsafe Casts
- `useAsyncStatus.ts:148,184,248,284`: `as AsyncStatusUpdate`, `as AsyncHistoryResponse` — unsafe casts on API responses. Low risk for intra-scenario boundary.
- `api.ts:920`: `JSON.parse(sseEvent.data) as StreamingEvent` — unsafe cast on SSE data. Low risk, has try/catch wrapper.
- `api.ts:274`: `res.json() as Promise<{...}>` — unsafe cast on health response. Low risk.

## Discovery/Lifecycle Findings
1. No hardcoded scenario `localhost:port` in production integration paths. Verified via `rg "localhost:[0-9]+"` — zero matches outside tests and resource defaults. Good.
2. `config.go:139`: Default Ollama URL `http://localhost:11434` is for a resource (not a scenario). Correctly uses env var override. Good.

## Fixes Applied (Proto Migration Session)
1. **Full proto migration for AgentManagerClient** (L0+L1/Critical): Replaced all `map[string]interface{}` parsing with proto types + `protojson`. Outbound requests now use correct snake_case keys and nested proto structure (`CreateTaskRequest{Task: ...}`, `CreateRunRequest{ProfileRef: ...}`). Inbound responses parsed via `protojson.Unmarshal` into generated types. Agent Inbox requests only its reconciled `agent-inbox/default` profile key; Agent Manager resolves the portable role and resource-native model.
2. **TranslateProtoEvent replaces TranslateEvent** (L1/High): New `TranslateProtoEvent(*domainpb.RunEvent)` uses typed oneof accessors instead of camelCase map lookups. Removed old `TranslateEvent(map[string]interface{})` and `NormalizeRunStatus()`.
3. **CheckAgentStatus returns `*domainpb.Run`** (L0/High): Changed signature from `(map[string]interface{}, error)` to `(*domainpb.Run, error)`. Reconciliation service updated to use proto field accessors.
4. **ProtoRunStatusToLocal helper** (L2/Medium): Maps proto `RunStatus` enum to local `RunStatus` string type. Replaces `NormalizeRunStatus()` string munging with canonical enum mapping.

## Fixes Applied (Previous Sessions)
1. **GetRunStatus status normalization** (L2/High): Status now derived from proto enum — no string normalization needed.
2. **StartAgentChat run-creation re-resolve** (L5/Medium): The second HTTP call in `StartAgentChat` (POST /api/v1/runs) re-resolves URL on connection error.
3. **PromptSyncService CRUD re-resolution** (L3/Medium): `CreateSkillInPromptManager`, `UpdateSkillInPromptManager`, `DeleteSkillInPromptManager`, and `RecordUsage` attempt URL re-resolution when `PromptManagerURL` is empty.
4. **AgentManagerClient URL re-resolution** (L5/Critical): Added `reResolveURL()`, `getBaseURL()`, `isConnectionError()`. All HTTP methods retry with re-resolved URL on connection failure.
5. **Centralized status normalization** (L2/High): Added `NormalizeRunStatus()`, `IsActiveRunStatus()`, `IsTerminalRunStatus()` to `agent_manager.go`. Updated `reconciliation.go` to use them.
6. **ProtocolHandler URL re-resolution** (L5/High): Added `URLResolver` field to `ProtocolHandlerConfig`. `Execute()` re-resolves on connection failure. `tool_registry.go` passes resolver when registering handlers.
7. **PromptSyncService URL re-resolution** (L5/High): Added `reResolveURL()` using `api-core/discovery`. `Sync()` re-resolves on connection failure and on empty URL.

## Remaining Risks
1. **Hand-written UI tool interfaces drift from proto** (L0/Medium): Six interfaces in `ui/src/lib/api.ts` duplicate proto types with missing fields (see UI↔API Findings above). New proto fields are silently unavailable to UI. Mitigation: replace with generated type imports when UI adopts `@vrooli/proto-types`.
2. **openrouter.go ConvertMessages type assertions** (L1/Low): Could panic on malformed internal data. Low risk since it's internal-only.
3. **UI `as` casts** (L1/Low): `useAsyncStatus.ts` and `api.ts` cast API responses without runtime validation. Low risk for intra-scenario boundary.
4. **Resilience package unused** (L5/Low): `resilience/` package has retry, circuit breaker, and fallback patterns fully implemented and tested, but no integration code imports or uses the package. Dead infrastructure until wired into adapters.

## Proper/Complete Gates
- [x] Contract safety - proto types used at all external API boundaries; internal maps are documented risk
- [x] Discovery/addressing safety - no hardcoded scenario ports; all scenario clients re-resolve on failure
- [x] Dependency parity - all 3 scenario deps declared in service.json match code usage
- [x] Envelope/status normalization - proto enum mapping via `ProtoRunStatusToLocal()`; all consumers use proto types
- [x] Runtime recovery - URL re-resolution on connection failure for all 3 scenario adapters, all code paths
- [ ] UI↔API contract safety - 6 hand-written tool interfaces duplicate proto types; no `fromJson`/`useProtoNames` in UI (acceptable for now since API serializes snake_case, but blocks adoption of generated types)
- [ ] Recovery tests - integration tests for restart/re-resolution scenarios (future work)
- [ ] Resilience wiring - connect resilience/ retry/circuit-breaker to integration adapters (future work)
