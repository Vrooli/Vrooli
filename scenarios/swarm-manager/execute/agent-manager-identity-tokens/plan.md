# Implementation Plan: Identity Token Injection and Verification

## 1. Purpose

Implement cryptographic identity tokens in agent-manager so that every agent run carries a verifiable proof of identity. Other scenarios (e.g., swarm-manager) can call back to agent-manager to verify who is making mutations, enabling trust-based attribution across the Vrooli ecosystem.

## 2. Required Reading

```bash
prompt-manager skill read api-steer seam-discovery-and-enforcement interoperability-steer
```

- **Research conclusions**: `swarm-manager backlog file-get --kind research --name agent-identity-standard --path conclusion.md`
- **Existing env injection**: `scenarios/agent-manager/api/internal/adapters/runner/env_utils.go`
- **Run executor**: `scenarios/agent-manager/api/internal/orchestration/run_executor.go`
- **Domain types**: `scenarios/agent-manager/api/internal/domain/types.go`
- **DB schema**: `scenarios/agent-manager/api/internal/database/schema.sql`
- **HTTP handlers**: `scenarios/agent-manager/api/internal/handlers/handlers.go`
- **Route registration**: `scenarios/agent-manager/api/main.go` (setupRoutes)

## 3. Problem Statement

Agent-manager creates and orchestrates agent runs but currently has no mechanism for those agents to cryptographically prove their identity to other services. The `CreatedBy` field exists but is a plain string with no verification. Other scenarios receiving requests from agents have no way to confirm which run/task/profile originated the request.

## 4. Scope

**In scope:**
- HMAC-SHA256 token generation at run creation time
- Secret key management (generate-on-first-startup, file-based persistence)
- `VROOLI_AGENT_IDENTITY_TOKEN` env var injection for all runs
- `POST /api/v1/identity/verify` REST endpoint
- Token revocation on run completion/cancellation/failure
- Unit and integration tests

**Out of scope:**
- CLI-core consumer library (separate item: `cli-core-identity-consumer`)
- Swarm-manager adoption (separate item: `swarm-manager-identity-adoption`)
- Secret rotation strategy (deferred — single-key is sufficient for v1)
- Multi-tenant / multi-server scenarios
- Token refresh mechanism (24h TTL is sufficient)

## 5. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `internal/orchestration/run_executor.go` | Run lifecycle; `SandboxEnvVars()` pattern to mirror |
| `internal/adapters/runner/env_utils.go` | `sanitizedBaseEnv()`, `appendEnvMap()` — env injection plumbing |
| `internal/domain/types.go` | Run, Task, AgentProfile domain structs |
| `internal/database/schema.sql` | SQLite schema — runs table |
| `internal/database/repository_run.go` | Run CRUD operations |
| `internal/handlers/handlers.go` | HTTP handler registration |
| `main.go` | Route registration in `setupRoutes()` |

### Existing Patterns to Follow
- **SandboxEnvVars()**: Returns `map[string]string` of env vars; called during env assembly. `IdentityEnvVars()` should mirror this pattern.
- **appendEnvMap()**: Merges extra env vars into the base env slice.
- **Handler factory**: Handlers are methods on a `Handler` struct; routes registered in `setupRoutes()`.
- **Error responses**: Use `domain.ErrorResponse` with code, message, userMessage, recovery, retryable fields.

## 6. Target End State

1. Every run (sandboxed or in-place) receives `VROOLI_AGENT_IDENTITY_TOKEN` in its environment
2. Token contains claims: `run_id`, `task_id`, `profile_key`, `scope_path`, `iat`, `exp`, `meta`
3. Any service can verify a token via `POST /api/v1/identity/verify` and receive the decoded claims
4. Tokens are automatically revoked when runs reach terminal status (complete, failed, cancelled)
5. Runs created before this feature continue working (no token = anonymous)

## 7. Implementation Strategy

### Phase 1: Token Infrastructure (New Package)
Create `internal/identity/` package:
- `token.go` — Claims struct, `Generate(claims, secret) → token string`, `Verify(token, secret) → claims, error`
- `secret.go` — `LoadOrCreateSecret(dataDir) → []byte` — reads from `<dataDir>/identity-secret.key`, generates 32-byte random key if missing
- Wire format: `base64url(json_claims).base64url(hmac_sha256(claims_bytes, secret))`

### Phase 2: Revocation Tracking
<!-- TBD — depends on decision d1 (in-memory vs DB column) -->

### Phase 3: Token Injection
- Add `IdentityEnvVars()` method on `RunExecutor` returning `map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": token}`
- Generate token during `Execute()` after run is created and has an ID
- Call in the same env assembly flow where `SandboxEnvVars()` is called
- Token generated with claims populated from `run`, `task`, and `profile`

### Phase 4: Verification Endpoint
- Add `POST /api/v1/identity/verify` route
- Handler: accepts `{"token": "..."}`, calls `identity.Verify()`, checks revocation, returns claims or 401
- New handler method on existing Handler struct (or new file `internal/handlers/identity.go`)

### Phase 5: Revocation Integration
- On run terminal status transitions (complete/failed/cancelled), mark token as revoked
- Verification endpoint checks revocation before returning success

### Phase 6: Tests
- Unit tests for `internal/identity/` (generate, verify, expired, tampered, revocation)
- Integration test: create run → extract env var → verify token → complete run → verify revoked

## 8. Contract Decisions

### Verification Endpoint
```
POST /api/v1/identity/verify
Content-Type: application/json

Request:  {"token": "base64url.base64url"}
Response (200): {"run_id": "...", "task_id": "...", "profile_key": "...", "scope_path": "...", "iat": 1234, "exp": 5678, "meta": {}}
Response (401): {"code": "TOKEN_INVALID", "message": "...", ...}
```

### Environment Variable
```
VROOLI_AGENT_IDENTITY_TOKEN=<base64url_claims>.<base64url_hmac>
```

### Claims Structure
```json
{
  "run_id": "uuid",
  "task_id": "uuid",
  "profile_key": "string",
  "scope_path": "string",
  "iat": 1234567890,
  "exp": 1234654290,
  "meta": {}
}
```

## 9. Testing Plan

### Unit Tests (`internal/identity/`)
- Generate token → verify succeeds with correct claims
- Tampered payload → verify fails
- Tampered signature → verify fails
- Expired token → verify fails
- Empty/malformed token → verify fails

### Integration Tests
- Create run → env contains `VROOLI_AGENT_IDENTITY_TOKEN`
- Extract token from env → POST to verify endpoint → 200 with correct claims
- Complete run → POST to verify endpoint → 401 (revoked)
- Cancel run → POST to verify endpoint → 401 (revoked)
- No token (legacy run) → system operates normally

### Handler Tests
- POST /api/v1/identity/verify with valid token → 200
- POST /api/v1/identity/verify with invalid token → 401
- POST /api/v1/identity/verify with missing body → 400
- POST /api/v1/identity/verify with expired token → 401

## 10. Rollout / Validation Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./... -timeout 300s` passes
- [ ] `gofumpt -l .` reports no formatting issues
- [ ] `golangci-lint run` passes
- [ ] Identity secret file created on first startup
- [ ] Token present in run env vars (verified via test)
- [ ] Verification endpoint returns correct claims
- [ ] Revocation works on run completion

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Secret file permissions too open | Medium | High | Set 0600 permissions on creation; document |
| In-memory revocation lost on restart | Medium | Low | Acceptable for v1 — expired tokens (24h) provide a ceiling; DB column is alternative |
| Token injection timing (run ID not yet available) | Low | Medium | Generate token after DB insert, before runner.Execute() |
| Performance of verification endpoint under load | Low | Low | HMAC verification is fast (~microseconds); revocation check is O(1) lookup |

## 12. Non-goals / Prohibited Patterns

- Do NOT distribute the HMAC secret to any consumer — verification is server-side only
- Do NOT use JWT libraries — use Go stdlib `crypto/hmac` + `crypto/sha256` only
- Do NOT add gRPC — REST only, consistent with existing API
- Do NOT break backward compatibility — missing token = anonymous, not error
- Do NOT add secret rotation in this item — single key is sufficient for v1

## 13. Definition of Done

- [ ] `internal/identity/` package exists with token generation and verification
- [ ] HMAC secret auto-generated and persisted on first startup
- [ ] `IdentityEnvVars()` injects `VROOLI_AGENT_IDENTITY_TOKEN` for all runs
- [ ] `POST /api/v1/identity/verify` endpoint operational
- [ ] Tokens revoked on run terminal status
- [ ] All tests pass (unit + integration)
- [ ] Code formatted with gofumpt, passes golangci-lint
- [ ] Scenario restarts successfully with `vrooli scenario restart agent-manager`
