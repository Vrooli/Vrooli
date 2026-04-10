# Implementation Plan: cli-core Identity Consumer Interface

## Required Reading

- `prompt-manager skill read cli-steer` — CLI architecture, cli-core patterns, cross-platform conventions
- `prompt-manager skill read test` — Testing patterns and conventions
- Review `packages/cli-core/cliutil/sandbox.go` — The pattern this implementation mirrors
- Review `packages/cli-core/cliutil/sandbox_test.go` — Test pattern to follow
- Review `packages/cli-core/cliutil/httpclient.go` — HTTPClient used for verification calls
- Review `packages/cli-core/cliutil/cliutil.go` — `APIBaseOptions` and `DetermineAPIBase` for URL resolution
- Review `scenarios/agent-manager/api/internal/identity/claims.go` — Canonical claims struct (source of truth for field names)
- Review `scenarios/agent-manager/api/internal/orchestration/service.go` — `IdentityVerifyResult` response shape

## Problem Statement

Scenario CLIs running inside agent-managed runs need to detect and verify agent identity tokens. The agent-manager already generates HMAC-SHA256 tokens and injects them as `VROOLI_AGENT_IDENTITY_TOKEN`, and exposes `POST /api/v1/identity/verify` for server-side verification. cli-core needs a consumer interface so any scenario can detect presence and verify tokens without each scenario reimplementing the HTTP call and parsing logic.

## Scope

### Acceptance Allow
- `packages/cli-core/**`

### What's In Scope
- `cliutil/identity.go` — Detection and verification functions
- `cliutil/identity_test.go` — Unit and integration-style tests
- Zero new dependencies (stdlib only + existing HTTPClient)

### What's Out of Scope
- Token generation/signing (agent-manager only)
- HMAC secret management (agent-manager only)
- Revocation tracking (agent-manager only)
- Changes to agent-manager or any scenario
- CLI command wiring (consuming scenarios do that themselves)

## Design Decisions (Settled)

### D1: Agent-manager base URL via env var `VROOLI_AGENT_MANAGER_API_BASE`
Agent-manager already injects env vars for runs. `VerifyIdentity()` reads `VROOLI_AGENT_MANAGER_API_BASE` internally using `DetermineAPIBase(APIBaseOptions{EnvVars: []string{"VROOLI_AGENT_MANAGER_API_BASE"}})`. No caller configuration needed.

### D2: VerifyIdentity is a zero-parameter method on IdentityEnv
`env.VerifyIdentity() (*VerifyResult, error)` — mirrors how `SandboxEnv.IsSandboxActive()` works. The token is encapsulated in the struct. Base URL comes from env var (D1).

### D3: Token-only IdentityEnv struct
`IdentityEnv` holds only the raw token string. All claims come from `VerifyResult` after calling the verification endpoint. This is honest — pre-verification, only the token is available.

### D4: Error semantics — transport vs. auth separation
- Transport errors (network down, timeout) → return Go `error`
- Auth errors (invalid/expired/revoked token) → return `*VerifyResult` with `Valid=false` and `Error` populated
- Callers use `if err != nil` for network issues and `if !result.Valid` for bad tokens

### D5: Fresh HTTPClient per call
`VerifyIdentity()` creates a new `HTTPClient` via `NewHTTPClient` with `APIBaseOptions{EnvVars: []string{"VROOLI_AGENT_MANAGER_API_BASE"}}` and a 10-second timeout on each invocation. Since verification is called at most once per CLI invocation, per-call construction is simpler than caching and avoids mutable global state. Tests use `t.Setenv("VROOLI_AGENT_MANAGER_API_BASE", server.URL)` to point at httptest servers.

### D6: 401 handling via APIError.RawResponse
When agent-manager returns 401, `HTTPClient.Do` wraps the response as `*APIError` (via `ParseAPIError`). `VerifyIdentity` type-asserts `errors.As(*APIError)` for status 401, then `json.Unmarshal(apiErr.RawResponse, &verifyResult)` to extract the structured `{valid: false, error: "..."}` body. This reuses existing HTTPClient infrastructure without duplicating request logic.

### D7: Exported constants for env var names
`EnvIdentityToken` and `EnvAgentManagerBase` are exported `const` values at the top of `identity.go`. Tests and consuming scenarios can reference them directly, improving maintainability over inline string literals (a refinement over the sandbox.go pattern).

## Approach

### Phase 1: Constants and Types

1. **Exported constants**:
   - `EnvIdentityToken = "VROOLI_AGENT_IDENTITY_TOKEN"`
   - `EnvAgentManagerBase = "VROOLI_AGENT_MANAGER_API_BASE"`

2. **`IdentityEnv` struct** — Single field: `Token string`

3. **`VerifiedClaims` struct** — Standalone type mirroring `identity.Claims` from agent-manager:
   - `RunID string` `json:"run_id"` (string, not UUID — cli-core has zero external deps)
   - `TaskID string` `json:"task_id"`
   - `ProfileKey string` `json:"profile_key"`
   - `ScopePath string` `json:"scope_path"`
   - `IssuedAt int64` `json:"iat"`
   - `ExpiresAt int64` `json:"exp"`
   - `Meta map[string]string` `json:"meta"`

4. **`VerifyResult` struct** — Mirrors `IdentityVerifyResult`:
   - `Valid bool` `json:"valid"`
   - `Claims *VerifiedClaims` `json:"claims"`
   - `RunStatus string` `json:"run_status"`
   - `Error string` `json:"error"`

### Phase 2: Detection Functions

1. **`DetectIdentity() IdentityEnv`** — Reads `EnvIdentityToken` from env, returns zero-value if absent
2. **`(env IdentityEnv) IsIdentityPresent() bool`** — Convenience method: returns `env.Token != ""`

### Phase 3: Verification Function

1. **`(env IdentityEnv) VerifyIdentity() (*VerifyResult, error)`**
   - Returns error immediately if `Token` is empty
   - Resolves agent-manager base URL via `DetermineAPIBase(APIBaseOptions{EnvVars: []string{EnvAgentManagerBase}})`
   - Returns error if base URL is empty (agent-manager not configured)
   - Creates a fresh `HTTPClient` with the resolved base options and 10-second timeout (D5)
   - POST `{"token": "<token>"}` to `/api/v1/identity/verify`
   - On success (200): deserialize response into `VerifyResult`, return it
   - On 401: type-assert `errors.As(*APIError)`, `json.Unmarshal(apiErr.RawResponse)` into `VerifyResult` with `Valid=false` (D6)
   - On other errors: return Go error (transport failure)

### Phase 4: Tests

1. **`TestDetectIdentity`** — Env var present/absent (mirrors `TestDetectSandbox`)
2. **`TestIsIdentityPresent`** — Token set/empty
3. **`TestVerifyIdentity_Success`** — httptest.Server returning 200 with valid claims
4. **`TestVerifyIdentity_InvalidToken`** — httptest.Server returning 401 with error JSON
5. **`TestVerifyIdentity_NetworkError`** — Closed server → Go error
6. **`TestVerifyIdentity_MalformedResponse`** — Server returns invalid JSON → Go error
7. **`TestVerifyIdentity_EmptyToken`** — Returns error without making HTTP call
8. **`TestVerifyIdentity_NoBaseURL`** — No env var set → returns error about missing config

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `packages/cli-core/cliutil/identity.go` | Create | Constants, `IdentityEnv`, `VerifiedClaims`, `VerifyResult`, `DetectIdentity()`, `IsIdentityPresent()`, `VerifyIdentity()` |
| `packages/cli-core/cliutil/identity_test.go` | Create | Unit tests for detection, httptest-based tests for verification |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Agent-manager verify endpoint contract changes | Low | Medium | Types mirror the current response shape; changes caught by integration tests |
| cli-core gains an external dependency accidentally | Low | High | PR review; go.mod has no external requires |
| Env var name changes | Very Low | Medium | Single exported constant, easy to update |
| HTTPClient internal construction leaks resources | Very Low | Low | Client is short-lived per call; Go's http.Client handles connection pooling |
| Timeout too short for slow networks | Low | Low | 10s is generous for a local loopback call; agent-manager runs on same host |

## Test Plan

1. `go test ./packages/cli-core/cliutil/ -run TestDetectIdentity -v` — Detection tests
2. `go test ./packages/cli-core/cliutil/ -run TestIsIdentityPresent -v` — Presence check tests
3. `go test ./packages/cli-core/cliutil/ -run TestVerifyIdentity -v` — All verification tests (success, invalid, network, malformed, empty token, no base URL)
4. `go build ./packages/cli-core/...` — Verify no new dependencies introduced
5. Verify `go.mod` unchanged (no new `require` lines)

## Implementation Notes

- Tests use `t.Setenv(EnvAgentManagerBase, server.URL)` to point at httptest servers — clean, no mocking needed
- `VerifyIdentity` creates an `HTTPClient` internally per call. For a verification call that happens once per CLI invocation, this is fine — no need to cache or pool
- JSON tags on `VerifiedClaims` must match the agent-manager response exactly: `run_id`, `task_id`, `profile_key`, `scope_path`, `iat`, `exp`, `meta`
- The 401 handling uses `errors.As` to extract `*APIError`, then `json.Unmarshal(apiErr.RawResponse, &result)` to parse the verification-specific response body
- Exported constants (`EnvIdentityToken`, `EnvAgentManagerBase`) improve maintainability over inline strings and can be referenced by consuming scenarios
