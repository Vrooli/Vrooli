# Research Conclusion: Verifiable Agent Identity Standard for Agent-Manager

## Research Question
How should agent-manager implement verifiable agent identity tokens so that spawned agents can cryptographically prove their identity to consuming scenarios, replacing the current unverified `CreatedBy` string attribution?

## Summary
Agent-manager will issue HMAC-SHA256 signed identity tokens to every spawned agent via the `VROOLI_AGENT_IDENTITY_TOKEN` env var. Tokens use a custom two-segment wire format (base64url claims + HMAC signature) built entirely on Go stdlib. Consuming scenarios verify tokens by calling a REST endpoint (`POST /api/v1/identity/verify`) on agent-manager, which validates the signature, checks expiry/revocation, and returns verified claims. The consumer Go interface in `packages/cli-core/cliutil/identity.go` mirrors the established `sandbox.go` pattern. Identity is orthogonal to sandboxing — both sandboxed and in-place runs receive tokens.

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
Agent-manager's API is built on `gorilla/mux` with standard `net/http` handlers. No gRPC service or dependency exists. Proto schemas under `packages/proto/schemas/agent-manager/` are used for JSON serialization only. **Decision: REST verification endpoint** (settled round 1, d3).

### Finding 2: cli-core has zero external dependencies
`packages/cli-core/go.mod` declares only `go 1.22` with no `require` block. JWT libraries (e.g., golang-jwt) would be the first external dep, propagating to every scenario. **Decision: custom HMAC-SHA256 using crypto/hmac + crypto/sha256 from stdlib** (settled round 1, d1).

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
- Token TTL should exceed run timeout to avoid mid-execution invalidation
- Revocation integrates with run completion/cancellation state transitions

### Finding 8: cli-core HTTP client is ready for verification calls
`cliutil/httpclient.go` provides `HTTPClient` with Bearer token injection and configurable timeouts (30s default). `cliutil/apiclient.go` wraps it with base URL resolution. Verification calls can reuse this infrastructure — `client.Post("/api/v1/identity/verify", tokenPayload)`.

## Design Decisions (Settled)

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
  "exp": 1711736100,
  "meta": {}
}
```

### Verification Model (Round 1, d2): Server-side verification
Tokens are opaque to consumers. The HMAC secret never leaves agent-manager. Consumers call the verification endpoint to validate tokens.

### Verification Protocol (Round 1, d3): REST
`POST /api/v1/identity/verify` — consistent with agent-manager's existing HTTP API and cli-core's HTTP client.

### Claims Structure (Round 1, d4): Minimal core + extensible metadata
Core claims: `run_id`, `task_id`, `profile_key`, `scope_path`, `iat`, `exp`. Extensible `meta` map for scenario-specific claims (initiative, spawning_scenario, etc.).

## Open Questions

### Token TTL Strategy (Round 2, d1)
Options under consideration:
- **A**: Static TTL = run timeout + 5 min buffer (recommended — simple, no refresh needed)
- **B**: Short TTL with heartbeat-based refresh (tighter security, more complexity)
- **C**: No expiry, revocation only (simplest but risky)

### Verification Endpoint Contract (Round 2, d2)
Options under consideration:
- **A**: Single verify endpoint returning full claims (recommended — simple, sufficient)
- **B**: Verify + separate introspection (/api/v1/identity/me) endpoint

### Verification Failure Handling (Round 2, d3)
Options under consideration:
- **A**: Fail open with warning, fall back to anonymous (recommended — matches backward-compat)
- **B**: Fail closed, reject unverifiable requests
- **C**: Cache-based fallback with short TTL

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
- Token TTL, verification endpoint contract, and failure handling are pending decisions (round 2)
- No revocation store design yet — needs specification of how agent-manager tracks revoked tokens (in-memory set vs DB column)
- No load testing analysis — verification endpoint latency under concurrent agents not measured
- Secret rotation strategy not yet addressed — how to rotate the HMAC signing key without invalidating active tokens

## Actions
1. **Create execute item**: `agent-manager-identity-tokens` — implement token generation, signing, injection, verification endpoint, and revocation in agent-manager API
2. **Create execute item**: `cli-core-identity-consumer` — implement `identity.go` in packages/cli-core with DetectIdentity/VerifyIdentity
3. **Create execute item**: `swarm-manager-identity-adoption` — update swarm-manager to verify agent identity on backlog mutations and execution attribution
4. **Future**: Secret rotation mechanism, audit logging of verification attempts, rate limiting on verification endpoint
