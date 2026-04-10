# Research Conclusion: Verifiable Agent Identity Standard for Agent-Manager

## Research Question
How should agent-manager implement verifiable agent identity tokens so that spawned agents can cryptographically prove their identity to consuming scenarios, replacing the current unverified `CreatedBy` string attribution?

## Summary
Agent-manager will issue HMAC-SHA256 signed identity tokens to every spawned agent via the `VROOLI_AGENT_IDENTITY_TOKEN` env var. Tokens use a custom two-segment wire format (base64url claims + HMAC signature) built entirely on Go stdlib. Consuming scenarios verify tokens by calling a REST endpoint (`POST /api/v1/identity/verify`) on agent-manager, which validates the signature, checks expiry/revocation, and returns verified claims. Tokens have a 24-hour default TTL to accommodate increasingly long-running agents. Verification failures are handled gracefully — scenarios fall back to anonymous attribution when agent-manager is unreachable. The consumer Go interface in `packages/cli-core/cliutil/identity.go` mirrors the established `sandbox.go` pattern. Identity is orthogonal to sandboxing — both sandboxed and in-place runs receive tokens.

## Methodology
- Examined the existing sandbox env var pattern in `packages/cli-core/cliutil/sandbox.go` (DetectSandbox, ScenarioInScope, ResolveMergedPath) as the ergonomic model to mirror
- Reviewed agent-manager's run executor (`internal/orchestration/run_executor.go`) and runner env injection (`internal/adapters/runner/claude_code.go`) to understand the injection point
- Surveyed current provenance tracking: unverified `CreatedBy` strings in tasks, `StartedBy`/`RequestedBy` in swarm-manager execution/activity records
- Checked existing dependencies: agent-manager uses gorilla/mux (HTTP only, no gRPC); cli-core has zero external dependencies; neither has JWT libraries
- Reviewed downstream execute items (`agent-manager-identity-tokens`, `swarm-manager-identity-adoption`) to understand planned consumption
- Analyzed run timeout (30 min default), heartbeat (15s interval, 5-min stale), and cancellation (StopRun/Terminator) to inform token lifecycle
- Reviewed cli-core HTTPClient (httpclient.go) and APIClient (apiclient.go) for verification endpoint consumption patterns

## Findings

### Finding 1: No gRPC exists — agent-manager is HTTP-only
Agent-manager's API is built on `gorilla/mux` with standard `net/http` handlers. No gRPC dependency or service definitions exist. Proto schemas under `packages/proto/schemas/agent-manager/` are used for JSON serialization only.

### Finding 2: cli-core has zero external dependencies
`packages/cli-core/go.mod` declares only `go 1.22` with no `require` block. JWT libraries (e.g., golang-jwt) would be the first external dep, propagating to every scenario. HMAC-SHA256 verification uses only `crypto/hmac` + `crypto/sha256` from stdlib.

### Finding 3: sandbox.go establishes the canonical consumer pattern
The sandbox detection pattern uses:
1. **Struct encapsulation**: `SandboxEnv{ID, Merged, Scope}` bundles related env vars
2. **Single detection entry point**: `DetectSandbox()` reads all env vars once
3. **Active check**: `IsSandboxActive()` validates required fields
4. **High-level helpers**: `ResolveScenarioPath()` abstracts complexity
5. **Zero-value safety**: Empty struct means "not in sandbox"

The identity standard mirrors this: `IdentityEnv` struct, `DetectIdentity()`, `IsIdentityPresent()`, and `VerifyIdentity()`.

### Finding 4: The injection point is well-defined
`RunExecutor.SandboxEnvVars()` (run_executor.go:923-941) returns a `map[string]string` that flows through `ExecuteRequest.Environment` → `buildEnv()` → `cmd.Env`. A new `IdentityEnvVars()` method on `RunExecutor` follows the same pattern. Unlike sandbox vars (only for sandboxed runs), identity vars are injected for ALL run modes.

### Finding 5: Current attribution is string-only and unverified
- `Task.created_by`: free-form string
- `execution.Record.StartedBy`: free-form string
- `PromptSkillVersion.CreatedBy`: free-form string
- No authentication middleware exists in agent-manager's HTTP handlers

Any process can claim any identity. The token standard lets consumers verify claims.

### Finding 6: Token size in env vars is not a practical concern
Linux env var limit is 128KB–2MB total per process. The proposed token format (base64url JSON claims + HMAC signature) is ~250-400 bytes. Negligible.

### Finding 7: Run lifecycle informs token TTL
- Default run timeout: 30 minutes (configurable per AgentProfile)
- Heartbeat interval: 15 seconds, stale threshold: 5 minutes
- Cancellation via `StopRun()` triggers state transition to `RunStatusCancelled`
- As AI agents become more capable, run durations will increase significantly beyond the current 30-minute default
- Token TTL should be generous to avoid mid-execution invalidation
- Revocation integrates with run completion/cancellation state transitions

### Finding 8: cli-core HTTP client is ready for verification calls
`cliutil/httpclient.go` provides `HTTPClient` with Bearer token injection and configurable timeouts (30s default). `cliutil/apiclient.go` wraps it with base URL resolution. Verification calls can reuse this infrastructure — `client.Post("/api/v1/identity/verify", tokenPayload)`.

## Design Decisions (All Settled)

### Token Format (Round 1, d1): Custom HMAC-SHA256
Wire format: `base64url(json_claims) + '.' + base64url(hmac_sha256(claims_bytes, server_secret))`

Two segments separated by a dot. Claims JSON structure:
```json
{
  "run_id": "uuid",
  "task_id": "uuid",
  "profile_key": "string",
  "scope_path": "string",
  "iat": 1711734000,
  "exp": 1711820400,
  "meta": {}
}
```

### Verification Model (Round 1, d2): Server-side verification
Tokens are opaque to consumers. The HMAC secret never leaves agent-manager. Consumers call the verification endpoint to validate tokens.

### Verification Protocol (Round 1, d3): REST
`POST /api/v1/identity/verify` — consistent with agent-manager's existing HTTP API and cli-core's HTTP client.

### Claims Structure (Round 1, d4): Minimal core + extensible metadata
Core claims: `run_id`, `task_id`, `profile_key`, `scope_path`, `iat`, `exp`. Extensible `meta` map for scenario-specific claims (initiative, spawning_scenario, etc.).

### Token TTL Strategy (Round 2, d1): 24-hour default TTL
Static TTL of 24 hours (86400 seconds). Rationale: agents are becoming more capable and will run for increasingly longer durations — the current 30-minute default timeout is already short and will grow. A 1-day TTL avoids mid-execution invalidation without requiring a refresh mechanism. Revocation on run cancellation/failure invalidates tokens early regardless of TTL. No refresh endpoint needed.

### Verification Endpoint Contract (Round 2, d2): Single verify endpoint
`POST /api/v1/identity/verify` with `{"token": "..."}` body. Returns 200 with full claims JSON on success, 401 on invalid/expired/revoked. Consumers get everything they need in one call. No separate introspection endpoint.

### Verification Failure Handling (Round 2, d3): Fail open with warning
If verification fails due to network issues (agent-manager unreachable), treat the request as anonymous/unverified and log a warning. Matches the backward-compatibility model where missing tokens = anonymous. Avoids blocking legitimate work due to transient infrastructure issues. Scenarios can optionally override this to fail closed if they require strict identity enforcement.

## Consumer Interface Design

**File**: `packages/cli-core/cliutil/identity.go`

```go
// IdentityEnv holds verified agent identity claims from the environment.
type IdentityEnv struct {
    Token      string // Raw VROOLI_AGENT_IDENTITY_TOKEN value
    RunID      string
    TaskID     string
    ProfileKey string
    ScopePath  string
    IssuedAt   int64
    ExpiresAt  int64
    Meta       map[string]string
}

// DetectIdentity reads VROOLI_AGENT_IDENTITY_TOKEN from the environment.
// Returns zero-value IdentityEnv if the var is absent.
func DetectIdentity() IdentityEnv { ... }

// IsIdentityPresent returns true if a token was found in the environment.
func (id IdentityEnv) IsIdentityPresent() bool { ... }

// VerifyIdentity calls agent-manager's verification endpoint and returns
// verified claims. Returns error if verification fails or agent-manager
// is unreachable.
func (id IdentityEnv) VerifyIdentity(client *HTTPClient) (*VerifiedClaims, error) { ... }
```

## Env Var Convention

| Variable | Purpose | Injected When |
|----------|---------|---------------|
| `VROOLI_AGENT_IDENTITY_TOKEN` | Signed identity token | All runs (sandboxed and in-place) |
| `VROOLI_SANDBOX_ID` | Sandbox UUID | Sandboxed runs only |
| `VROOLI_SANDBOX_MERGED` | Overlay merged path | Sandboxed runs only |
| `VROOLI_SANDBOX_SCOPE` | Scope path | Sandboxed runs with scope |

Identity is orthogonal to sandboxing. A run can have both identity + sandbox vars, or identity only (in-place runs).

## Backward Compatibility
- Runs created before this standard: no `VROOLI_AGENT_IDENTITY_TOKEN` in env
- `DetectIdentity()` returns zero-value → `IsIdentityPresent()` returns false
- Consumers fall back to anonymous/operator attribution (existing `CreatedBy` strings)
- Identity verification is opt-in: scenarios that don't import identity.go are unaffected
- No breaking changes to existing env vars or APIs

## Limitations
- No revocation store design yet — needs specification of how agent-manager tracks revoked tokens (in-memory set vs DB column on runs table)
- No load testing analysis — verification endpoint latency under concurrent agents not measured
- Secret rotation strategy not yet addressed — how to rotate the HMAC signing key without invalidating active tokens
- Fail-open default is appropriate for current single-operator deployment but may need revisiting for multi-tenant scenarios

## Actions

### Action 1: Create backlog item — Implement identity token generation and verification in agent-manager
- **Kind**: execute
- **Title**: Implement agent identity token generation, signing, injection, and verification endpoint in agent-manager API
- **Description**: Add HMAC-SHA256 token signing with server-side secret. Implement `IdentityEnvVars()` on RunExecutor to inject `VROOLI_AGENT_IDENTITY_TOKEN` for all runs. Add `POST /api/v1/identity/verify` REST endpoint that validates signature, checks expiry/revocation, and returns verified claims. Add revocation tracking (mark tokens invalid on run cancellation/completion). Default TTL: 24 hours.
- **Initiative**: swarm-manager-feature-parity
- **Priority**: 2
- **Effort**: M

### Action 2: Create backlog item — Implement identity consumer interface in cli-core
- **Kind**: execute
- **Title**: Implement identity.go consumer interface in packages/cli-core
- **Description**: Add `cliutil/identity.go` with `IdentityEnv` struct, `DetectIdentity()`, `IsIdentityPresent()`, and `VerifyIdentity()` functions. Mirror the established `sandbox.go` ergonomic pattern. Zero external dependencies — use existing `HTTPClient` for verification calls. Handle verification failures gracefully (fail open with warning by default).
- **Initiative**: swarm-manager-feature-parity
- **Priority**: 2
- **Effort**: S
- **Depends on**: execute/agent-manager-identity-tokens

### Action 3: Create backlog item — Update swarm-manager to verify agent identity
- **Kind**: execute
- **Title**: Update swarm-manager to verify agent identity on backlog mutations and execution attribution
- **Description**: Import cli-core identity package. On backlog mutations and execution attribution, verify `VROOLI_AGENT_IDENTITY_TOKEN` against agent-manager before trusting `StartedBy`/`CreatedBy` fields. Fall back to anonymous attribution if token absent or verification fails.
- **Initiative**: swarm-manager-feature-parity
- **Priority**: 2
- **Effort**: S
- **Depends on**: execute/cli-core-identity-consumer

### Action 4: Future work (not immediate backlog items)
- Secret rotation mechanism for HMAC signing key
- Audit logging of verification attempts
- Rate limiting on verification endpoint
- Multi-tenant identity scoping if deployment model changes
