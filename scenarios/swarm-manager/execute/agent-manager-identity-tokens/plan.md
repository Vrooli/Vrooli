# Implementation Plan: Identity Token Injection and Verification in Agent-Manager

## 1. Purpose

Implement cryptographic identity tokens in agent-manager so that spawned agents can prove their identity to consuming scenarios. This replaces unverified `CreatedBy` string attribution with HMAC-SHA256 signed tokens injected via `VROOLI_AGENT_IDENTITY_TOKEN` and verified through a REST endpoint.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring api-steer test seam-discovery-and-enforcement
```

- **Research conclusion**: `swarm-manager backlog file-get --kind research --name agent-identity-standard --path conclusion.md`
- **Codebase entry points**:
  - `scenarios/agent-manager/api/internal/orchestration/run_executor.go` — `SandboxEnvVars()` pattern to mirror
  - `scenarios/agent-manager/api/internal/adapters/runner/env_utils.go` — `sanitizedBaseEnv()`, `appendEnvMap()`
  - `scenarios/agent-manager/api/internal/handlers/handlers.go` — route registration
  - `scenarios/agent-manager/api/internal/domain/types.go` — Run domain model
  - `scenarios/agent-manager/api/internal/database/schema.sql` — SQLite schema

## 3. Problem Statement

Agent-manager spawns agent runs (sandboxed and in-place) that interact with other scenarios. Currently, there is no way for a consuming scenario to verify that a request actually came from a specific agent run. The `CreatedBy` field is a free-form string anyone can spoof. This blocks swarm-manager's need for reliable provenance tracking on backlog mutations and execution attribution.

## 4. Scope

### In Scope
- HMAC-SHA256 token generation and signing within agent-manager
- Server-side HMAC secret management (generate on first startup, persist)
- `IdentityEnvVars()` method on RunExecutor for env var injection
- `POST /api/v1/identity/verify` REST endpoint
- Token revocation tracking on run cancellation/completion
- Unit and integration tests

### Out of Scope
- Consumer interface in `packages/cli-core` (separate backlog item: `execute/cli-core-identity-consumer`)
- Swarm-manager adoption (separate backlog item: `execute/swarm-manager-identity-adoption`)
- Secret rotation mechanism (future work)
- Rate limiting on verification endpoint (future work)
- Multi-tenant identity scoping (future work)

## 5. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `internal/orchestration/run_executor.go` | RunExecutor with `SandboxEnvVars()` pattern (line 934) |
| `internal/adapters/runner/env_utils.go` | `sanitizedBaseEnv()`, `appendEnvMap()` |
| `internal/handlers/handlers.go` | Route registration via `RegisterRoutes()` (line 96) |
| `internal/domain/types.go` | Run, Task, AgentProfile domain types |
| `internal/database/schema.sql` | SQLite schema definitions |
| `internal/database/connection.go` | SQLite path resolution (line 73-105) |
| `internal/repository/interface.go` | Repository interfaces |

### Existing Patterns
- **Env var injection**: `SandboxEnvVars()` returns `map[string]string`, flows through `ExecuteRequest.Environment` → `buildEnv()` → `cmd.Env`
- **Route registration**: `RegisterRoutes(r *mux.Router)` with gorilla/mux
- **Handler pattern**: Thin HTTP handlers that delegate to orchestration service
- **Test infrastructure**: SQLite-backed repos with seeded entities, `testFixtures` struct, mock sandbox/runner providers
- **Data persistence**: SQLite DB at configurable path (env vars: `AM_SQLITE_PATH` → `DATABASE_URL` → `SQLITE_DATABASE_PATH` → `VROOLI_DATA`)

## 6. Target End State

After implementation:
1. Every agent run (sandboxed and in-place) receives `VROOLI_AGENT_IDENTITY_TOKEN` in its environment
2. Agent-manager persists an HMAC signing secret in its data directory
3. `POST /api/v1/identity/verify` accepts a token, validates signature + expiry + revocation, returns verified claims
4. Tokens are automatically revoked when runs complete or are cancelled
5. All existing runs continue working without tokens (backward compatible)
6. Comprehensive test coverage for token lifecycle

## 7. Implementation Strategy

### Phase 1: Signing Infrastructure
1. Create `internal/identity/` package with:
   - `secret.go` — HMAC secret generation, loading, and persistence
   - `token.go` — Token generation (claims → signed token string) and verification (token string → claims)
   - `claims.go` — Claims struct definition
2. Secret stored as a file in agent-manager's data directory (alongside SQLite DB)
3. Secret generated on first startup using `crypto/rand` (32 bytes)

### Phase 2: Token Generation & Injection
1. Add `IdentityEnvVars()` method to RunExecutor (mirrors `SandboxEnvVars()`)
2. Generate token when run transitions to starting phase
3. Claims populated from Run + Task + AgentProfile domain objects:
   - `run_id`: Run.ID
   - `task_id`: Run.TaskID
   - `profile_key`: AgentProfile.ProfileKey
   - `scope_path`: Task.ScopePath
   - `iat`: current time
   - `exp`: current time + 86400 (24h default TTL)
   - `meta`: empty map (extensible)
4. Inject `VROOLI_AGENT_IDENTITY_TOKEN` via existing env pipeline

### Phase 3: Revocation Tracking
1. Add `identity_token_hash` and `identity_token_revoked` columns to runs table
2. Store SHA-256 hash of token (not the token itself) for revocation lookups
3. On run completion/cancellation, set `identity_token_revoked = true`
4. Verification checks revocation status via run lookup

### Phase 4: Verification Endpoint
1. Register `POST /api/v1/identity/verify` in `RegisterRoutes()`
2. Handler accepts `{"token": "..."}` JSON body
3. Verification steps: parse token → validate HMAC signature → check expiry → check revocation
4. Returns 200 with full claims JSON on success, 401 on failure
5. No authentication middleware required (the token IS the authentication)

### Phase 5: Tests
1. Unit tests for `internal/identity/` package (token generation, signing, verification, expiry, tampering)
2. Unit tests for `IdentityEnvVars()` method
3. Integration tests for verification endpoint (valid token, expired token, revoked token, malformed token)
4. Integration test for full flow: create run → extract env var → verify token → complete run → verify revocation

## 8. Contract Decisions

### Wire Format
```
base64url(json_claims) + '.' + base64url(hmac_sha256(claims_bytes, secret))
```

### Claims JSON
```json
{
  "run_id": "uuid-string",
  "task_id": "uuid-string",
  "profile_key": "string",
  "scope_path": "string",
  "iat": 1711734000,
  "exp": 1711820400,
  "meta": {}
}
```

### Verification Endpoint
- **Route**: `POST /api/v1/identity/verify`
- **Request**: `{"token": "base64url.base64url"}`
- **Success (200)**: `{"valid": true, "claims": { ...full claims... }}`
- **Failure (401)**: `{"valid": false, "error": "token expired" | "token revoked" | "invalid signature" | "malformed token"}`

### Env Var
- Name: `VROOLI_AGENT_IDENTITY_TOKEN`
- Injected: All runs (sandboxed and in-place)
- Absent: Backward compatible — consumers treat as anonymous

## 9. Testing Plan

| Test | Type | What It Validates |
|------|------|-------------------|
| Token round-trip | Unit | Generate → sign → verify returns original claims |
| Expired token | Unit | Token with past `exp` is rejected |
| Tampered token | Unit | Modified claims with original signature is rejected |
| Tampered signature | Unit | Original claims with wrong signature is rejected |
| Secret persistence | Unit | Secret survives load/save cycle |
| IdentityEnvVars sandboxed | Unit | Sandboxed run gets token in env map |
| IdentityEnvVars in-place | Unit | In-place run gets token in env map |
| Verify endpoint valid | Integration | POST with valid token returns 200 + claims |
| Verify endpoint expired | Integration | POST with expired token returns 401 |
| Verify endpoint revoked | Integration | POST with revoked token returns 401 |
| Verify endpoint malformed | Integration | POST with garbage returns 401 |
| Full lifecycle | Integration | Create run → get token → verify → complete run → verify revoked |

## 10. Rollout/Validation Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./... -timeout 300s` passes (all existing + new tests)
- [ ] `gofumpt -w .` applied to all new files
- [ ] `golangci-lint run` passes
- [ ] Manual smoke test: start agent-manager, create a run, verify token appears in env, call verify endpoint

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Secret file permissions too open | Token forgery | Set file permissions to 0600 on creation |
| Secret lost on data dir wipe | All active tokens invalid | Acceptable — tokens are short-lived, runs restart cleanly |
| Verification endpoint adds latency | Slower cross-scenario calls | Single DB lookup per verify, sub-ms for SQLite |
| Token in env var visible to child processes | Token leakage | Acceptable — child processes are the agent itself; same trust boundary |
| Schema migration on existing DBs | Startup failure | Use `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` pattern |

## 12. Non-goals / Prohibited Patterns

- Do NOT add JWT library dependencies — use stdlib `crypto/hmac` + `crypto/sha256` only
- Do NOT expose the HMAC secret via any API or env var
- Do NOT add authentication middleware to the verify endpoint — the token is self-authenticating
- Do NOT modify existing `SandboxEnvVars()` — add new `IdentityEnvVars()` alongside it
- Do NOT break existing runs that lack identity tokens

## 13. Definition of Done

1. All runs (sandboxed and in-place) receive `VROOLI_AGENT_IDENTITY_TOKEN` in their environment
2. `POST /api/v1/identity/verify` correctly validates tokens and returns claims
3. Tokens are revoked when runs complete or are cancelled
4. All tests in the testing plan pass
5. `go build`, `go test`, `gofumpt`, `golangci-lint` all pass
6. Backward compatible — existing runs without tokens continue working
