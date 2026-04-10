# Implementation Plan: Identity Token Injection and Verification in Agent-Manager

## 1. Purpose

Implement cryptographic identity tokens in agent-manager so that spawned agents can prove their identity to consuming scenarios. This replaces unverified `CreatedBy` string attribution with HMAC-SHA256 signed tokens injected via `VROOLI_AGENT_IDENTITY_TOKEN` and verified through a REST endpoint.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring api-steer test seam-discovery-and-enforcement security
```

- **Research conclusion**: `swarm-manager backlog file-get --kind research --name agent-identity-standard --path conclusion.md`
- **Codebase entry points**:
  - `scenarios/agent-manager/api/internal/orchestration/run_executor.go` — `SandboxEnvVars()` pattern (line 934), `MergedEnvVars()` (line 957), `Execute()` lifecycle
  - `scenarios/agent-manager/api/internal/adapters/runner/env_utils.go` — `sanitizedBaseEnv()` (line 21), `appendEnvMap()` (line 40)
  - `scenarios/agent-manager/api/internal/handlers/handlers.go` — `RegisterRoutes()` (line 96), route registration pattern
  - `scenarios/agent-manager/api/internal/domain/types.go` — Run domain model, run phases (line 1314-1344)
  - `scenarios/agent-manager/api/internal/database/schema.sql` — SQLite schema, runs table (line 59-96)
  - `scenarios/agent-manager/api/internal/database/connection.go` — `sqliteDSN()` data dir resolution (line 73)

## 3. Problem Statement

Agent-manager spawns agent runs (sandboxed and in-place) that interact with other scenarios. Currently, there is no way for a consuming scenario to verify that a request actually came from a specific agent run. The `CreatedBy` field is a free-form string anyone can spoof. This blocks swarm-manager's need for reliable provenance tracking on backlog mutations and execution attribution.

## 4. Scope

### In Scope
- HMAC-SHA256 token generation and signing within agent-manager
- Server-side HMAC secret management (generate on first startup, persist as file)
- Token generation during run execution with hash persistence in runs table
- `VROOLI_AGENT_IDENTITY_TOKEN` env var injection for ALL runs via `MergedEnvVars()`
- `POST /api/v1/identity/verify` REST endpoint returning claims + run status
- Token lifecycle tied to run state transitions
- Unit and integration tests

### Out of Scope
- Consumer interface in `packages/cli-core` (separate item: `execute/cli-core-identity-consumer`)
- Swarm-manager adoption (separate item: `execute/swarm-manager-identity-adoption`)
- Secret rotation mechanism (future work)
- Rate limiting on verification endpoint (future work)
- Multi-tenant identity scoping (future work)

## 5. Current Technical Context

### Key Files
| File | Role | Key Lines |
|------|------|-----------|
| `internal/orchestration/run_executor.go` | RunExecutor with env injection, lifecycle hooks | SandboxEnvVars:934, MergedEnvVars:957, handleSuccessfulCompletion:1040, handleFailure:1090, handleCancellation:1116 |
| `internal/adapters/runner/env_utils.go` | Env pipeline: sanitize → append | sanitizedBaseEnv:21, appendEnvMap:40 |
| `internal/adapters/runner/claude_code.go` | Claude Code runner buildEnv | buildEnv:716 |
| `internal/handlers/handlers.go` | Route registration via gorilla/mux | RegisterRoutes:96, requestIDMiddleware:180 |
| `internal/domain/types.go` | Run, Task, AgentProfile domain types | RunPhase constants:1314-1344 |
| `internal/database/schema.sql` | SQLite schema definitions | runs table:59-96 |
| `internal/database/connection.go` | SQLite DSN resolution (data dir path) | sqliteDSN:73-105 |
| `internal/repository/interface.go` | Repository interfaces | |

### Existing Patterns
- **Env var injection**: `SandboxEnvVars()` returns `map[string]string` → `MergedEnvVars()` merges custom + sandbox vars → flows through `ExecuteRequest.Environment` → per-runner `buildEnv()` → `cmd.Env`. Identity vars integrate at the `MergedEnvVars()` level alongside sandbox vars.
- **Route registration**: `RegisterRoutes(r *mux.Router)` with gorilla/mux, `requestIDMiddleware` on all routes
- **Handler pattern**: Thin HTTP handlers delegating to orchestration service
- **Run lifecycle**: Phases transition through Queued → Initializing → ... → Executing → ... → Completed/Failed/Cancelled. Terminal handlers: `handleSuccessfulCompletion()`, `handleFailure()`, `handleCancellation()`
- **Test infrastructure**: SQLite-backed repos with `setupTestDB(t)`, `testFixtures` struct, mock providers
- **Data persistence**: SQLite at configurable path (priority: `AM_SQLITE_PATH` → `DATABASE_URL` → `SQLITE_DATABASE_PATH` → `VROOLI_DATA` → `~/.vrooli/data/sqlite/databases/agent-manager.db`)

## 6. Target End State

After implementation:
1. Every agent run (sandboxed and in-place) receives `VROOLI_AGENT_IDENTITY_TOKEN` in its environment
2. Agent-manager persists an HMAC signing secret as `identity-secret.key` (0600 permissions) in its data directory
3. `POST /api/v1/identity/verify` accepts a token, validates HMAC signature + expiry, returns verified claims with run status
4. Token hashes stored in runs table enable post-completion identity lookups
5. All existing runs continue working without tokens (backward compatible)
6. Comprehensive test coverage for token lifecycle

## 7. Implementation Strategy

### Phase 1: Signing Infrastructure
1. Create `internal/identity/` package:
   - `secret.go` — HMAC secret generation (32 bytes via `crypto/rand`), loading from file, persistence with 0600 permissions
   - `token.go` — Token generation (claims → signed token string) and verification (token string → claims)
   - `claims.go` — Claims struct definition (`run_id`, `task_id`, `profile_key`, `scope_path`, `iat`, `exp`, `meta`)
2. Secret stored as `identity-secret.key` in same directory as SQLite DB (resolved via `sqliteDSN()` parent dir)
3. **Startup behavior**: Fail startup with a clear error message if the secret cannot be loaded or created. If the file exists but is unreadable (permissions), corrupted (partial write), or the data directory is read-only, agent-manager refuses to start. This is the safest posture for a security component — silent degradation would mask real problems and lead to confusion when tokens are silently absent.

### Phase 2: Token Generation & Injection
1. **Generation point**: Generate token in `RunExecutor.Execute()` during the phase transition to `RunPhaseExecuting`, where DB access is already available. Store the SHA-256 hash of the token in the runs table. Pass the token string to the runner via env vars. This keeps the DB write in the orchestration layer where it belongs.
2. Claims populated from Run + Task + AgentProfile domain objects:
   - `run_id`: Run.ID
   - `task_id`: Run.TaskID
   - `profile_key`: AgentProfile.ProfileKey
   - `scope_path`: Task.ScopePath
   - `iat`: current time
   - `exp`: current time + 86400 (24h default TTL)
   - `meta`: empty map (extensible)
3. Inject `VROOLI_AGENT_IDENTITY_TOKEN` via `MergedEnvVars()` — identity vars merge alongside sandbox vars, both override custom vars. The token is generated in `Execute()` and stored on the RunExecutor instance, then `MergedEnvVars()` reads it as a simple accessor. No per-runner changes needed since all three runners (claude_code.go, opencode_runner.go, codex_runner.go) receive the merged env via `ExecuteRequest.Environment`.

### Phase 3: Run Status Tracking for Tokens
1. Add columns to existing runs table via `ALTER TABLE ... ADD COLUMN`:
   - `identity_token_hash TEXT` — SHA-256 hash of the generated token (for lookup during verification)
   - `identity_token_revoked_at TEXT` — timestamp when token was explicitly revoked (NULL = not revoked)
2. Schema migration runs in `initSchema()` using `ALTER TABLE ... ADD COLUMN` (SQLite supports this idempotently with IF NOT EXISTS pattern via pragma check)
3. On run completion/cancellation/failure, the terminal handlers (`handleSuccessfulCompletion()`, `handleFailure()`, `handleCancellation()`) update `identity_token_revoked_at`
4. **Verification behavior**: The verify endpoint always returns claims + `run_status` for any valid-signature, non-expired token. Revocation status is communicated via the `run_status` field (`active`, `completed`, `cancelled`, `failed`), letting consumers decide their own trust policy. Token expiry (24h TTL) is the only hard cutoff that produces a 401. This matches the user's requirement that identity should always be retrievable via the token while the token hasn't expired.

### Phase 4: Verification Endpoint
1. Register `POST /api/v1/identity/verify` in `RegisterRoutes()`
2. Handler accepts `{"token": "..."}` JSON body
3. Verification steps: parse token → validate HMAC signature → check expiry → look up run by token hash → return claims + run status
4. **Response contract**:
   - **200**: `{"valid": true, "claims": {...}, "run_status": "active|completed|cancelled|failed"}` — for any valid-signature, non-expired token. Run status lets consumers decide trust level.
   - **401**: `{"valid": false, "error": "..."}` — for invalid signature, expired, or malformed tokens only
5. No authentication middleware on this endpoint — the token IS the authentication

### Phase 5: Tests
1. Unit tests for `internal/identity/` package:
   - Token round-trip (generate → sign → verify returns original claims)
   - Expired token rejected
   - Tampered claims with original signature rejected
   - Tampered signature with original claims rejected
   - Secret persistence (load/save cycle)
2. Unit tests for token generation integration:
   - Token appears in env vars for sandboxed runs
   - Token appears in env vars for in-place runs
   - Token hash stored in runs table
3. Integration tests for verification endpoint:
   - Valid token returns 200 + claims + run_status "active"
   - Expired token returns 401
   - Completed run token returns 200 + run_status "completed"
   - Cancelled run token returns 200 + run_status "cancelled"
   - Malformed token returns 401
4. Integration test for full lifecycle:
   - Create run → extract token from env → verify token (active) → complete run → verify token (completed status, still 200) → wait for expiry → verify (401 expired)

## 8. Contract Decisions

### Wire Format (settled — research)
```
base64url(json_claims) + '.' + base64url(hmac_sha256(claims_bytes, secret))
```

### Claims JSON (settled — research)
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

### Verification Endpoint (settled — round 2 d1)
- **Route**: `POST /api/v1/identity/verify`
- **Request**: `{"token": "base64url.base64url"}`
- **Success (200)**: `{"valid": true, "claims": { ...full claims... }, "run_status": "active|completed|cancelled|failed"}`
- **Failure (401)**: `{"valid": false, "error": "token expired" | "invalid signature" | "malformed token"}`

### Env Var (settled — research)
- Name: `VROOLI_AGENT_IDENTITY_TOKEN`
- Injected: All runs (sandboxed and in-place) via `MergedEnvVars()`
- Absent: Backward compatible — consumers treat as anonymous

### Schema Addition (settled — round 1 d2)
```sql
ALTER TABLE runs ADD COLUMN identity_token_hash TEXT;
ALTER TABLE runs ADD COLUMN identity_token_revoked_at TEXT;
```

## 9. Testing Plan

| Test | Type | What It Validates |
|------|------|-------------------|
| Token round-trip | Unit | Generate → sign → verify returns original claims |
| Expired token | Unit | Token with past `exp` is rejected |
| Tampered token | Unit | Modified claims with original signature is rejected |
| Tampered signature | Unit | Original claims with wrong signature is rejected |
| Secret persistence | Unit | Secret survives load/save cycle |
| Secret file permissions | Unit | File created with 0600 permissions |
| IdentityEnvVars sandboxed | Unit | Sandboxed run gets token in env map |
| IdentityEnvVars in-place | Unit | In-place run gets token in env map |
| Token hash stored | Unit | Token hash written to runs table on generation |
| Verify endpoint valid | Integration | POST with valid token returns 200 + claims + run_status "active" |
| Verify endpoint expired | Integration | POST with expired token returns 401 |
| Verify endpoint completed run | Integration | POST with completed run's token returns 200 + run_status "completed" |
| Verify endpoint cancelled run | Integration | POST with cancelled run's token returns 200 + run_status "cancelled" |
| Verify endpoint malformed | Integration | POST with garbage returns 401 |
| Full lifecycle | Integration | Create run → verify active → complete → verify completed (200) → expiry → 401 |

## 10. Rollout/Validation Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./... -timeout 300s` passes (all existing + new tests)
- [ ] `gofumpt -w .` applied to all new files
- [ ] `golangci-lint run` passes
- [ ] Manual smoke test: start agent-manager, create a run, verify token appears in env, call verify endpoint

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Secret file permissions too open | Token forgery | Set file permissions to 0600 on creation; verify on load |
| Secret lost on data dir wipe | All active tokens invalid | Acceptable — tokens are short-lived (24h), runs restart cleanly |
| Verification endpoint adds latency | Slower cross-scenario calls | Single DB lookup per verify, sub-ms for SQLite |
| Token in env var visible to child processes | Token leakage | Acceptable — child processes are the agent itself; same trust boundary |
| Schema migration on existing DBs | Startup failure | Use `ALTER TABLE ADD COLUMN` with existence check (query pragma_table_info) |
| Concurrent token generation race | Duplicate/missing hashes | SQLite WAL mode + single-writer handles this; RunExecutor already serializes per-run |
| Secret file unreadable/corrupted at startup | Agent-manager won't start | Intentional — fail loudly so admin fixes the issue rather than silently running without identity |

## 12. Non-goals / Prohibited Patterns

- Do NOT add JWT library dependencies — use stdlib `crypto/hmac` + `crypto/sha256` only
- Do NOT expose the HMAC secret via any API or env var
- Do NOT add authentication middleware to the verify endpoint — the token is self-authenticating
- Do NOT modify existing `SandboxEnvVars()` — add identity logic alongside it in `MergedEnvVars()`
- Do NOT break existing runs that lack identity tokens
- Do NOT modify per-runner `buildEnv()` methods — identity flows through `MergedEnvVars()` → `ExecuteRequest.Environment`

## 13. Definition of Done

1. All runs (sandboxed and in-place) receive `VROOLI_AGENT_IDENTITY_TOKEN` in their environment
2. `POST /api/v1/identity/verify` correctly validates tokens and returns claims with run status for non-expired tokens (200 with run_status for completed/cancelled runs, 401 only for expired/invalid)
3. Token hashes stored in runs table for verification lookups
4. Agent-manager fails startup with clear error if HMAC secret cannot be loaded or created
5. All tests in the testing plan pass
6. `go build`, `go test`, `gofumpt`, `golangci-lint` all pass
7. Backward compatible — existing runs without tokens continue working
