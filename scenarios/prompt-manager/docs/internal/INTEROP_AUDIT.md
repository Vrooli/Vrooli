# Interoperability Audit — prompt-manager

**Last Updated**: 2026-02-13 (re-audited: no changes needed)

## Dependency Inventory

| Dependency | Type | Declared | Required | Protocol | Status |
|---|---|---|---|---|---|
| agent-manager | scenario | Yes | No | HTTP/JSON (protojson) | Healthy |
| postgres | resource | Yes | Yes | SQL (lib/pq) | Healthy |
| qdrant | resource | Yes | No | HTTP REST | Healthy (graceful degrade) |
| ollama | resource | Yes | No | HTTP REST | Healthy (graceful degrade) |

## Contract Stack Assessment

| Layer | Status | Notes |
|---|---|---|
| L0 Schema | Partial | agent-manager has proto schemas; prompt-manager uses hand-written Go types (acceptable — only subset of Run/Task fields needed) |
| L1 Serialization | Fixed | JSON tags now use snake_case matching agent-manager's `protojson UseProtoNames=true` output |
| L2 Envelope/Status | Good | Centralized in `heartbeat/status.go`; `IsTerminalStatus`/`IsFailedStatus`/`IsCancelledStatus` with tests |
| L3 Discovery | Good | Per-request `discovery.ResolveScenarioURLDefault` in `heartbeat/client.go:332` |
| L4 Lifecycle | Good | `service.json` declares agent-manager (optional), postgres (required), qdrant/ollama (optional) |
| L5 Runtime Recovery | Good | `doRequestWithRetry` (3 attempts, 500ms/1s/2s backoff); `RunRegistry.Recover()` for restart; 15min timeout on WaitForRun |

## Issues Resolved

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
- **Hand-written types vs proto imports**: prompt-manager only uses a small subset of agent-manager's Run/Task fields. Hand-written types are acceptable here, but JSON tags must match protojson output format. If more fields are needed in the future, consider importing generated proto types directly.

## Remaining Risks

1. **Status vocabulary drift**: If agent-manager adds new terminal statuses (e.g., `RUN_STATUS_TIMED_OUT`), they won't be recognized until `heartbeat/status.go` is updated. Source of truth: `packages/proto/schemas/agent-manager/v1/domain/types.proto`.
2. **Proto type safety gap**: prompt-manager deserializes agent-manager responses via `encoding/json` into hand-written structs, not via `protojson` into generated types. This means new/renamed fields require manual sync. Acceptable for now given the narrow interface (5 API calls).
3. **Resource URL resolution**: Ollama and Qdrant URLs come from environment variables, not from discovery. This is correct for resources (not scenarios) but means URL changes require process restart.
4. **`needs_review` timeout behavior**: If a heartbeat run reaches `RUN_STATUS_NEEDS_REVIEW` (approval required), `WaitForRun` will poll until the 15-minute timeout fires and then mark the run as failed. Acceptable because heartbeat runs use `RUN_MODE_IN_PLACE` and should not normally require approval.

## Completion Gates

- [x] All runtime scenario dependencies declared in service.json
- [x] No duplicated status vocabularies
- [x] Transient failure resilience for cross-scenario calls
- [x] No hardcoded resource addresses
- [x] JSON serialization tags match protojson output format
- [x] Error field deserialization verified
- [x] Status normalization centralized and tested
- [x] Audit documented
