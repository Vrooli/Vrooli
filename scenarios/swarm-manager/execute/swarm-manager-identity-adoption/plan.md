# Implementation Plan: Adopt Agent Identity Standard in Swarm-Manager

## 1. Purpose

Integrate the agent identity standard into swarm-manager so that backlog mutations, execution attribution, and agent activity records carry verified provenance. This enables distinguishing operator-created from agent-created items and tracing changes back to specific agent runs.

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer seam-discovery-and-enforcement test
```

- Review `packages/cli-core/cliutil/identity.go` — `DetectIdentity()`, `VerifyIdentity()`, `IdentityEnv`, `VerifyResult`, `VerifiedClaims`
- Review `packages/cli-core/cliutil/identity_test.go` — Test patterns for identity detection/verification
- Review `packages/cli-core/cliutil/httpclient.go` — `HTTPClient` struct, `DoWithContext()`, header injection points
- Review `packages/cli-core/cliutil/apiclient.go` — `APIClient.Request()`, `AuthHeaders()`
- Review `scenarios/swarm-manager/api/internal/execution/model.go` — `Record.StartedBy`, `CreateRequest`
- Review `scenarios/swarm-manager/api/internal/agentactivity/types.go` — `Record.RequestedBy`, `Spec.normalized()`, context pattern
- Review `scenarios/swarm-manager/api/internal/backlog/types.go` — `BacklogItem` struct, `SpawnedFrom` field
- Review `scenarios/swarm-manager/api/internal/backlog/store.go` — `SaveItem()` read-modify-write pattern, field preservation
- Review `scenarios/swarm-manager/cli/helpers.go` — `requestV1()`, `requestMultipartV1()`, `AuthHeaders()` iteration
- Review `scenarios/swarm-manager/api/main.go` — `loggingMiddleware` global middleware pattern, `setupRoutes()`
- Review `scenarios/swarm-manager/api/internal/testutil/helpers.go` — shared test utilities

## 2. Problem Statement

Swarm-manager currently has unverified free-form `started_by` / `requested_by` strings on execution records and agent activity entries. These are hardcoded to values like `"swarm-manager"` or `"swarm-manager-ui"`. When agents create or modify backlog items via the CLI, there is no way to:

1. Distinguish agent-created items from human-created ones
2. Trace which specific agent run produced a change
3. Verify that the claimed identity is authentic

The cli-core identity consumer interface (`execute/cli-core-identity-consumer`, now completed) provides `DetectIdentity()` and `VerifyIdentity()`. This item adopts those primitives across swarm-manager's CLI, API, and UI.

## 3. Scope

### In Scope (acceptance_allow)
- `scenarios/swarm-manager/api/**` — API middleware, provenance extraction, storage
- `scenarios/swarm-manager/cli/**` — Identity detection, header injection
- `scenarios/swarm-manager/ui/**` — Provenance display in execution cards and backlog views

### Out of Scope
- Token generation/signing (agent-manager responsibility)
- Changes to `packages/cli-core/` (already completed)
- Changes to `packages/proto/` (existing proto fields sufficient)
- Changes to agent-manager
- Backlog item access control based on identity (future work)

## 4. Current Technical Context

### CLI Side
- `cli/helpers.go`: `requestV1()` delegates to `a.core.APIClient.Request()` → `HTTPClient.DoWithContext()`. No hook for extra headers.
- `cli/helpers.go`: `requestMultipartV1()` manually constructs HTTP requests and iterates `AuthHeaders()` to set headers.
- `cli/cmd_execution.go`: Already has `--started-by` flag (default: `"swarm-manager"`).
- `cli/app.go`: `App` struct has `core *cliapp.ScenarioApp` and `globalDry bool`.
- **Key constraint**: `HTTPClient` only sets `Authorization` and `X-Dry-Run` headers. It has no `ExtraHeaders` map or `SetExtraHeaders()` method. Identity header injection requires wrapping the transport (see Phase 1).

### API Side
- `api/internal/execution/model.go`: `Record.StartedBy` (string) and `CreateRequest.StartedBy` (string) already exist.
- `api/internal/execution/service_queue.go`: `QueueBacklog()` defaults `StartedBy` to `"swarm-manager"` if empty (line 109).
- `api/internal/agentactivity/types.go`: `Record.RequestedBy` and `Spec.RequestedBy` (string) — defaults to `"swarm-manager"` in `Spec.normalized()` (line 128).
- `api/internal/backlog/types.go`: `BacklogItem` has `SpawnedFrom` but no `CreatedBy` field.
- `api/internal/backlog/store.go`: Read-modify-write pattern in `SaveItem()` — reads existing JSON, merges known fields, preserves unknown fields.
- `api/main.go`: Global `loggingMiddleware` shows the middleware registration pattern (`s.router.Use(loggingMiddleware)` at line 102).
- No identity/auth middleware exists.

### UI Side
- `ui/src/components/execution/execution-card.tsx`: Displays `item.startedBy` as plain text `"by {startedBy}"` (line 122).
- `ui/src/services/execution-service.ts`: `CreateExecutionRequest` already includes optional `startedBy`.
- Reusable badge patterns: `InitiativeBadge` (pill with color), `AgentRunningBadge` (animated dot + label).

### cli-core Identity API (Dependency — Completed)
- `DetectIdentity() IdentityEnv` — reads `VROOLI_AGENT_IDENTITY_TOKEN` from env
- `(env IdentityEnv) IsIdentityPresent() bool`
- `(env IdentityEnv) VerifyIdentity() (*VerifyResult, error)` — calls agent-manager `POST /api/v1/identity/verify`
- `VerifyResult{Valid, Claims *VerifiedClaims, RunStatus, Error}`
- `VerifiedClaims{RunID, TaskID, ProfileKey, ScopePath, IssuedAt, ExpiresAt, Meta}`

## 5. Target End State

1. **CLI**: When `VROOLI_AGENT_IDENTITY_TOKEN` is present, the CLI automatically includes it in API requests via an `X-Agent-Identity-Token` header. No manual `--started-by` needed for agent runs.
2. **API**: Identity middleware extracts the token from the header, verifies it via a `Verifier` interface (backed by cli-core's `VerifyIdentity()`), and injects a `Provenance` struct into the request context. Downstream handlers use provenance to populate `started_by`, `requested_by`, and `created_by` fields.
3. **Backlog items**: `spec.json` gains an immutable `created_by` field set at creation time — structured object: `{"type":"operator"}` or `{"type":"agent","run_id":"...","task_id":"...","profile_key":"..."}`. Immutability enforced at the store level.
4. **Execution records**: `started_by` field carries structured provenance as a string (backward-compatible: `"operator"` or `"agent:<profile_key>/<run_id>"`).
5. **Agent activity**: `requested_by` automatically populated from context provenance when not explicitly set by handlers.
6. **UI**: Inline provenance badges with icons visually distinguish operator vs agent attribution. Agent-created items show bot icon + profile_key, clickable to originating run.
7. **Graceful fallback**: Missing/invalid token = `{"type":"operator"}` attribution. Requests never fail due to identity issues.

## 6. Implementation Strategy

### Phase 1: CLI Identity Detection & Header Injection

**Files:**
- `scenarios/swarm-manager/cli/app.go` — Call `DetectIdentity()` at startup, store on `App`
- `scenarios/swarm-manager/cli/helpers.go` — Inject `X-Agent-Identity-Token` header via custom RoundTripper
- `scenarios/swarm-manager/cli/identity_transport.go` (new) — `identityTransport` wrapping `http.DefaultTransport`

**Approach — Custom RoundTripper (Workshop d1-r2: option A confirmed):**
1. In `App` initialization, call `cliutil.DetectIdentity()` and store the `IdentityEnv` on the `App` struct
2. Create `identityTransport` implementing `http.RoundTripper` that wraps the default transport and injects `X-Agent-Identity-Token` on every request when the token is present
3. Set this transport on the `http.Client` used by `APIClient` before any requests are made
4. This transparently covers both `requestV1()` and `requestMultipartV1()` paths
5. When identity is present, auto-set `--started-by` default to `"agent:<profile_key>"` instead of `"swarm-manager"`

**Why RoundTripper**: Go-idiomatic pattern. Works transparently for all HTTP paths without modifying cli-core. The `HTTPClient` uses a standard `*http.Client` internally — wrapping its `Transport` injects headers at the transport layer.

### Phase 2: API Identity Middleware

**Files:**
- `scenarios/swarm-manager/api/internal/identity/` (new package) — `middleware.go`, `provenance.go`, `middleware_test.go`
- `scenarios/swarm-manager/api/main.go` — Register identity middleware

**Approach — Interface-based Verifier (Workshop d2-r2: option A confirmed):**
1. Define `Provenance` struct: `{Type string, RunID string, TaskID string, ProfileKey string}`
2. Define `Verifier` interface: `Verify(token string) (*cliutil.VerifyResult, error)` — production implementation wraps cli-core's `VerifyIdentity()`, tests use a stub
3. Create `Middleware(verifier Verifier) func(http.Handler) http.Handler` that:
   a. Reads `X-Agent-Identity-Token` header
   b. If absent → set `Provenance{Type: "operator"}` in context, continue
   c. If present → call `verifier.Verify(token)`
   d. On success (Valid=true) → set `Provenance{Type: "agent", ...claims}` in context
   e. On failure → log warning, set `Provenance{Type: "operator"}` (fail open)
4. Provide `FromContext(ctx) Provenance` helper
5. Register as global middleware in `main.go` after `loggingMiddleware`: `s.router.Use(identity.Middleware(realVerifier))` (Workshop d3-r1: option A confirmed — global middleware, verify once per request)

### Phase 3: Backlog Item Provenance

**Files:**
- `scenarios/swarm-manager/api/internal/backlog/types.go` — Add `CreatedBy` field to `BacklogItem`
- `scenarios/swarm-manager/api/internal/backlog/store.go` — Persist and protect `created_by` in `SaveItem()`
- `scenarios/swarm-manager/api/internal/backlog/handler_create.go` — Set `CreatedBy` from context provenance

**Approach — Store-level immutability (Workshop d3-r2: option A confirmed):**
1. Add `CreatedBy *Provenance` field to `BacklogItem` (pointer so omitempty works for existing items without the field)
2. In `SaveItem()` read-modify-write: read existing `created_by` from disk first; if already set, preserve the existing value regardless of what the struct says — defense in depth, the invariant is impossible to violate regardless of caller
3. In create handler, extract provenance from context via `identity.FromContext(r.Context())` and set `CreatedBy`
4. Update handlers never need to touch `created_by` — the store protects it

### Phase 4: Execution & Activity Record Provenance

**Files:**
- `scenarios/swarm-manager/api/internal/execution/service_queue.go` — Use context provenance for `StartedBy`
- `scenarios/swarm-manager/api/internal/agentactivity/types.go` — Enhance `Spec.normalized()` to read provenance from context

**Approach — Context-based propagation (Workshop d4-r2: option A confirmed):**
1. In `QueueBacklog()`, if `req.StartedBy` is empty, check context provenance before defaulting to `"swarm-manager"`. If provenance is agent type, format as `"agent:<profile_key>/<run_id>"`.
2. In agent activity `Spec.normalized()`, if `RequestedBy` is empty, check context provenance. If agent type, format similarly. Otherwise keep `"swarm-manager"` default.
3. Internal API callers that hardcode `RequestedBy: "swarm-manager"` stay unchanged — these are server-initiated, not agent-initiated.

**Note on `started_by` format (Workshop d2-r1: option A confirmed):** Kept as string for backward compatibility. Convention: `"operator"` or `"agent:<profile_key>/<run_id>"`. No proto changes needed — old records keep their existing values, UI parses the string.

### Phase 5: UI Provenance Display

**Files:**
- `scenarios/swarm-manager/ui/src/components/execution/execution-card.tsx` — Replace plain text with provenance badge
- `scenarios/swarm-manager/ui/src/components/shared/identity-badge.tsx` (new) — Reusable provenance badge component

**Approach — Inline badge with icon (Workshop d4-r1: option A confirmed):**
1. Create `IdentityBadge` component following the `InitiativeBadge` pattern:
   - Parse `startedBy` / `createdBy` to determine operator vs agent
   - Operator: user icon + "operator" label
   - Agent: bot icon + `profile_key` label, clickable to run if `run_id` available
2. Replace `execution-card.tsx` line 122 (`<span>by {item.startedBy}</span>`) with `<IdentityBadge value={item.startedBy} />`
3. For `created_by` on backlog views: if the field exists in the response, show the badge; otherwise omit (backward compatible)

### Phase 6: Tests

**Files:**
- `scenarios/swarm-manager/api/internal/identity/middleware_test.go` — Middleware unit tests
- `scenarios/swarm-manager/cli/identity_transport_test.go` — RoundTripper unit tests

**Test Cases:**
1. Middleware: no header → `Provenance{Type: "operator"}` in context
2. Middleware: valid token → `Provenance{Type: "agent", RunID: ..., ProfileKey: ...}` in context
3. Middleware: invalid token (Valid=false) → `Provenance{Type: "operator"}` + warning log
4. Middleware: verification error (network) → `Provenance{Type: "operator"}` + warning log
5. CLI transport: identity present → `X-Agent-Identity-Token` header set on outgoing requests
6. CLI transport: identity absent → no identity header
7. Backlog create with agent provenance → `created_by` persisted in spec.json
8. Backlog update → `created_by` immutable (store preserves original)
9. Execution create with provenance → `started_by` has structured agent format

**Testing approach (Workshop d2-r2: option A confirmed):** Interface-based `Verifier` with stub implementation in tests. No running agent-manager or httptest server needed for unit tests. Clean testing seam that matches existing testutil patterns.

## 7. Contract Decisions

### HTTP Header
- Header name: `X-Agent-Identity-Token`
- Value: raw token string (opaque to transport layer)

### Provenance JSON Shape (in spec.json `created_by`) — Workshop d1-r1: option A confirmed
```json
{"type": "operator"}
// or
{"type": "agent", "run_id": "...", "task_id": "...", "profile_key": "..."}
```
Structured object chosen over compact string for rich queryability and direct display without parsing.

### Execution `started_by` String Format (backward compatible) — Workshop d2-r1: option A confirmed
- Operator: `"operator"` or legacy value (e.g., `"swarm-manager"`)
- Agent: `"agent:<profile_key>/<run_id>"`
String convention chosen to avoid proto breaking changes.

### Fail-Open Behavior
- Missing token → operator attribution, no warning
- Invalid/expired token → operator attribution + warning log
- Agent-manager unreachable → operator attribution + warning log
- Never return 401/403 for identity issues

## 8. Testing Plan

### Unit Tests
- Identity middleware: 4 test cases via mock `Verifier` interface
- CLI identity transport: 2 test cases (RoundTripper injects/omits header)
- Provenance serialization/deserialization round-trip
- Store-level `created_by` immutability

### Integration Tests
- Full flow: CLI with identity env → API middleware → backlog creation → verify spec.json
- Full flow: execution creation with identity → verify `started_by` format
- Backward compatibility: old records without `created_by` load correctly

## 9. Rollout/Validation Checklist

- [ ] CLI builds and passes tests
- [ ] API builds and passes tests
- [ ] UI builds successfully
- [ ] Existing backlog items load without errors (no `created_by` = nil)
- [ ] New backlog items get `created_by = operator` when no identity present
- [ ] Agent-initiated backlog items get `created_by` with agent claims
- [ ] Execution records show structured `started_by` for new runs
- [ ] Legacy execution records display correctly in UI
- [ ] Agent-manager being down does not break any swarm-manager operations

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Agent-manager down during verification | Medium | Low | Fail open — operator attribution, warning log |
| Backward compatibility with existing records | Medium | Medium | Nil/missing `created_by` treated as operator; legacy `started_by` strings displayed as-is |
| HTTPClient transport wrapping breaks assumptions | Low | Medium | Unit test the RoundTripper; verify it composes with existing Authorization header |
| Performance overhead of verification HTTP call | Low | Low | Single call per request; agent-manager is localhost; 10s timeout |
| cli-core VerifyIdentity API changes | Low | Medium | Pin to current interface; integration tests catch drift |
| Proto changes needed for structured provenance | Low | Medium | Avoided — string encoding for `started_by`, JSON for `created_by` in spec.json |

## 11. Non-goals / Prohibited Patterns

- Do NOT add authentication/authorization to swarm-manager (identity ≠ auth)
- Do NOT parse tokens directly — always use cli-core's `VerifyIdentity()`
- Do NOT fail requests when identity is missing or invalid
- Do NOT modify `packages/cli-core/` or `packages/proto/`
- Do NOT add backward-compatibility shims for old `started_by` values — display as-is
- Do NOT add identity to internal server-initiated operations (only external CLI/API requests)

## 12. Definition of Done

1. CLI auto-detects identity and injects token header via custom RoundTripper
2. API middleware verifies tokens via `Verifier` interface and sets context provenance
3. New backlog items have immutable `created_by` in spec.json (store-enforced)
4. New execution records have structured `started_by`
5. Agent activity records auto-populate `requested_by` from context provenance
6. UI shows provenance badges with operator/agent distinction
7. All tests pass (unit + integration)
8. Graceful degradation when agent-manager is unavailable
9. Existing data loads and displays without errors
