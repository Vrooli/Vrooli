# Interoperability Audit — prompt-manager

**Last Updated**: 2026-11-16

## Dependency Inventory

| Dependency | Type | Declared | Required | Protocol | Status |
|---|---|---|---|---|---|
| agent-manager | scenario | Yes | No | Generated Connect RPC plus narrow REST owner operations | Healthy |
| postgres | resource | Yes | Yes | SQL (lib/pq) | Healthy |
| qdrant | resource | Yes | No | HTTP REST | Healthy (graceful degrade) |
| ollama | resource | Yes | No | resource-ollama gateway CLI | Healthy (graceful degrade) |

## Contract Stack Assessment

| Layer | Status | Notes |
|---|---|---|
| L0 Schema | Good | Agent Manager requests and responses for implemented RPCs use generated Go protobuf messages. Prompt Manager retains small local projections for heartbeat-domain state and converts only at the adapter boundary. |
| L1 Transport/Serialization | Partial | Health, profile/run reads, task/run creation, effort-board reads, and investigation listing use generated Connect clients and ProtoJSON. StartInvestigation remains REST pending equivalent authorization-approval semantics review. |
| L2 Envelope/Status | Good | Centralized in `heartbeat/status.go`; `IsTerminalStatus`/`IsFailedStatus`/`IsCancelledStatus` with tests |
| L3 Discovery | Good | Connect clients resolve Agent Manager on each call/each retry through `discovery.ResolveScenarioURLDefault`. |
| L4 Lifecycle | Good | `service.json` declares agent-manager (optional), postgres (required), qdrant/ollama (optional) |
| L5 Runtime Recovery | Good | `doRequestWithRetry` (3 attempts, 500ms/1s/2s backoff); `RunRegistry.Recover()` for restart; 15min timeout on WaitForRun |

## Issues Resolved

| # | Issue | Resolution | Date |
|---|---|---|---|
| 10 | Heartbeat integration used REST/protojson despite implemented Agent Manager RPCs | Migrated health, profile listing, task/run creation, run reads/list/events, effort-board reads, and typed investigation listing to generated Connect procedures. CreateTask/CreateRun preserve approval-pending state from the typed `AuthorizationApprovalPending` error detail. UI effort-board client already used the generated Agent Manager Connect service. | 2026-11-16 |
| 11 | `GetEffortEnrollment` hand-built an HTTP POST to a generated procedure path | Replaced with the generated Go Connect client and generated request/response messages. | 2026-11-16 |

| # | Issue | Resolution | Date |
|---|---|---|---|
| 1 | agent-manager not declared in service.json | Added `dependencies.scenarios.agent-manager` (required: false) | 2026-02-13 |
| 2 | Terminal status strings duplicated in 3 places | Centralized in `heartbeat/status.go` with `IsTerminalStatus`/`IsFailedStatus` | 2026-02-13 |
| 3 | No retry/backoff on agent-manager calls | Added `doRequestWithRetry` — 3 attempts, 500ms/1s/2s backoff, transport+5xx only | 2026-02-13 |
| 4 | Hardcoded qdrant fallback `localhost:6333` | Removed fallback; empty QDRANT_URL now disables AI search (consistent with ollama) | 2026-02-13 |
| 5 | No INTEROP_AUDIT.md | Created this document | 2026-02-13 |
| 6 | **Run struct JSON tags used camelCase** | Fixed to snake_case matching agent-manager's `protojson UseProtoNames=true` output | 2026-02-13 |
| 7 | **`Run.Error` field tag `"error"` never matched proto's `error_msg`** | Fixed to `json:"error_msg"` — error messages from failed runs were previously silently lost | 2026-02-13 |
| 8 | **Request types used camelCase JSON tags** | Fixed `CreateRunRequest`, `ProfileRef`, `EnsureProfileRequest`, `AgentProfile` to use snake_case | 2026-02-13 |
| 9 | Missing `IsCancelledStatus` helper | Added `IsCancelledStatus` with test coverage | 2026-02-13 |

## Design Decisions

- **StopRun excluded from retry**: Time-sensitive operation; retrying a stop could mask cancellation state.
- **agent-manager required: false**: Core prompt CRUD is functional without heartbeats; only the heartbeat subsystem needs agent-manager.
- **Qdrant fallback removed**: AI search already gracefully degrades; a hardcoded localhost URL would silently connect to an unrelated qdrant instance.
- **Local projections vs proto imports**: heartbeat persistence and scheduling use a narrow local task/run projection. The generated messages are authoritative on the wire; conversion stays at the Connect adapter.

## Remaining Risks

1. **StartInvestigation authorization parity:** Prompt Manager's typed investigation admission still uses REST. Review whether the endpoint's authorization middleware can return approval-pending and add the equivalent Connect detail mapping before migrating this mutation.
2. **REST operations without matching RPCs:** `GetTask`, task cancel/delete, run stop/continue, investigation-apply, identity verification, and run listing with investigation relationship filters remain REST because Agent Manager has no matching `AgentManagerService` procedure or request fields. These should be revisited when the owner adds the missing contract; schema messages without service methods do not count as Connect coverage.
3. **Status vocabulary drift:** local heartbeat state classification still uses the generated Agent Manager status vocabulary in `heartbeat/status.go`; review it when the owner adds terminal statuses.
4. **Local heartbeat projections:** Prompt Manager keeps narrow local task/run projections because its heartbeat domain and persistence API consume a subset with JSON-compatible opaque fields. The wire request/response is now the generated protobuf contract at the Connect adapter.
5. **Resource configuration resolution:** Qdrant URL and Ollama gateway command come from environment variables, not scenario discovery. This is correct for resources; configuration changes require process restart.
6. **`needs_review` timeout behavior:** If a heartbeat run reaches `RUN_STATUS_NEEDS_REVIEW`, `WaitForRun` polls until the 15-minute timeout fires and then marks it failed. Heartbeat runs normally use `RUN_MODE_IN_PLACE` and should not require approval.

## Completion Gates

- [x] All runtime scenario dependencies declared in service.json
- [x] No duplicated status vocabularies
- [x] Transient failure resilience for cross-scenario calls
- [x] No hardcoded resource addresses
- [x] Generated Connect clients used for all operations with implemented procedures
- [x] Error field deserialization verified
- [x] Status normalization centralized and tested
- [x] Audit documented
