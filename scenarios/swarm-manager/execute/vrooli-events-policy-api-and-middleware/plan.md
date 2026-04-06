# Implementation Plan: Policy CRUD API And Receiver-Side Enforcement Middleware

## Required Reading

```bash
prompt-manager skill read api-steer boundary-of-responsibility-enforcement
swarm-manager backlog file-get --kind execute --name vrooli-events-core-runtime --path plan.md
```

## 1. Purpose

Extend the existing vrooli-events scenario with three-layer receiver-side enforcement middleware (access control with specificity scoring, token bucket rate limiting, and circuit breaking) and convert the policy push mechanism from incremental events to full snapshot broadcast.

## 2. Problem Statement

The vrooli-events core runtime (completed) includes a working policy CRUD API, SQLite store, evaluator, broadcaster, and receiver-side middleware — but all of these currently only handle **access control** rules. Rate limiting and circuit breaking rule types can be stored but are not enforced. The middleware (`internal/middleware/policy.go`) skips non-access rules entirely (line 177). Additionally:

- Access control uses manual priority ordering rather than the spec's auto-computed specificity scoring
- Policy push broadcasts incremental events rather than full snapshots
- No token bucket rate limiter state management exists
- No circuit breaker state machine (closed/open/half-open) with threshold tracking exists
- **Bug**: Middleware endpoint matching (line 189) uses direct string equality instead of glob pattern matching via `internal/match/` package

This item completes the policy enforcement layer so all three rule types are functional end-to-end.

## 3. Scope

### In Scope
- **Access control specificity scoring**: Auto-compute priority from pattern specificity on rule create/update
- **Fix endpoint matching bug**: Use `internal/match/` glob matching in middleware (consistent with evaluator)
- **Token bucket rate limiting middleware**: Per-rule in-memory state with capacity + refill_rate enforcement
- **Circuit breaker middleware**: 3-state machine (closed/open/half-open) with configurable threshold, cooldown, and auto-reset
- **Policy snapshot push**: Replace incremental broadcaster with full-snapshot SSE push on any rule change
- **Middleware chaining**: Three layers (access → rate limit → circuit breaker) as composable http.Handler middleware
- **Tests**: Unit tests for each middleware layer, integration tests for the full chain

### Out of Scope
- Discovery package sender-side policy cache (separate item: `execute/discovery-event-emission-and-policy-cache`)
- Analytics UI (separate item: `execute/vrooli-events-analytics-ui`)
- Authentication/authorization on policy API endpoints
- UI of any kind
- Distributed rate limiting (single-process in-memory is sufficient for receiver-side)
- Server-side evaluator changes (stays access-only for the /evaluate dry-run endpoint)

## 4. Dependencies

| Dependency | Kind | Status | What It Provides |
|---|---|---|---|
| `execute/vrooli-events-core-runtime` | execute | **completed** | Full policy CRUD API, SQLite store, evaluator, broadcaster, middleware skeleton |

No blockers.

## 5. Settled Decisions

### From Round 1

| ID | Decision | Choice | Rationale |
|---|---|---|---|
| R1-d1 | Schema: flat columns vs rule_data JSON | **Keep flat columns** | Already working, type-safe, queryable. Slight denormalization acceptable at current scale. |
| R1-d2 | Access control priority | **Auto-computed specificity scoring** | Per spec: exact=3pts, prefix glob=2pts, wildcard=1pt, summed across source+target+action (max 9). Ties broken by earliest creation time. Score stored in priority column, computed on create/update. |
| R1-d3 | Rate limiter state storage | **In-memory with sync.Mutex** | Fastest path, no disk I/O. State resets on restart (acceptable: just means full bucket). |
| R1-d4 | Policy push mechanism | **Full snapshot push** | Atomic replacement, no drift risk. Payload small at current scale (dozens of rules). |
| R1-d5 | Middleware scope | **Receiver-side only** | Three-layer enforcement in receiver-side middleware. Server-side evaluator stays access-only for /evaluate endpoint. |

### From Round 2

| ID | Decision | Choice | Rationale |
|---|---|---|---|
| R2-d1 | Middleware layer architecture | **Separate files per layer** | Each file owns its layer: `policy_access.go`, `policy_ratelimit.go`, `policy_circuit.go`. Shared types/config stay in `policy.go`. Each layer has distinct state and logic; separate files are easier to test, review, and reason about independently. |
| R2-d2 | Circuit breaker failure detection | **HTTP 5xx responses only** | Standard industry practice. Server errors indicate downstream is unhealthy; client errors (4xx) are caller mistakes, not service failures. Simple and predictable. |
| R2-d3 | Middleware chaining pattern | **Standard nested http.Handler wrapping** | Idiomatic Go: `accessMW(rateLimitMW(circuitBreakerMW(handler)))`. Each layer calls `next.ServeHTTP` only if it passes. Simple, composable, follows stdlib patterns. Each layer is independently testable. |
| R2-d4 | State cleanup on snapshot push | **Prune on snapshot** | On every snapshot update, each stateful layer iterates its state maps and removes entries whose rule IDs are absent from the new snapshot. O(n) where n = state entries, negligible at current scale. Clean and correct. |

## 6. Current Technical Context

### Existing Policy Infrastructure (from core-runtime)
- **API handlers**: `api/handlers_policy.go` — Full CRUD (create, list, get, update, delete), violations, evaluate, circuit breaker override, policy subscribe SSE
- **Routes**: `api/routes.go` — All policy endpoints wired
- **Store**: `internal/policy/sqlite.go` — SQLite with policy_rules, policy_violations, circuit_breaker_overrides tables. ListRules ordered by `priority DESC, id ASC`.
- **Types**: `internal/policy/policy.go` — Rule, Violation, Decision, CircuitState, Store interface
- **Evaluator**: `internal/policy/evaluator.go` — Server-side access-only evaluation (queries DB, used by POST /evaluate). Filters to RuleTypeAccess at line 31-36.
- **Broadcaster**: `internal/policy/broadcaster.go` — Incremental event broadcaster (created/updated/deleted), non-blocking with 64-capacity channels
- **Middleware**: `internal/middleware/policy.go` — Receiver-side access-only middleware with cached rules, SSE update, health info. Filters to RuleTypeAccess at line 177.
- **Match**: `internal/match/` — Segment-aware glob matching (separator ".", * = one segment, ** = one or more)
- **Headers**: `internal/headers/` — X-Source-Scenario extraction/injection
- **Fallback**: `internal/fallback/` — fail-open/fail-closed mode when vrooli-events unreachable

### Key Observations
- Middleware has 18 existing tests (`internal/middleware/policy_test.go`) including explicit test that rate_limit rules are ignored (line 316)
- Evaluator has 6 tests (access rules only)
- Broadcaster tests confirm incremental event model
- Handler validation already supports all three rule types
- Circuit breaker override API exists in handlers but has no enforcement backing
- SSE stale detection: 60+ seconds without updates + disconnected = stale cache

## 7. Target End State

The receiver-side middleware becomes a composable three-layer HTTP middleware chain:
1. **Access control** (`policy_access.go`): Specificity-scored rule matching with glob pattern matching on all fields, most-specific-wins, 403 on deny
2. **Rate limiting** (`policy_ratelimit.go`): Token bucket per-rule with in-memory state (sync.Mutex-guarded map), 429 responses with Retry-After header
3. **Circuit breaking** (`policy_circuit.go`): 3-state machine per route key, HTTP 5xx as failure signal, 503 responses during open state with Retry-After, auto-reset on cooldown

Policy changes trigger full snapshot push via SSE, and receivers atomically replace their cached rules. Stateful layers (rate limiter, circuit breaker) prune orphaned state entries not present in the new snapshot.

## 8. Implementation Strategy

All workshop decisions are settled. No pending decisions remain.

### Phase 1: Access Control Specificity Scoring + Bug Fix
**Files**: `internal/policy/sqlite.go`, `internal/middleware/policy_access.go` (extracted from `policy.go`)

1. Add `ComputeSpecificity(source, target, action string) int` function:
   - Per-pattern scoring: exact match = 3pts, prefix glob (e.g. `swarm-*`) = 2pts, wildcard (`*`) = 1pt
   - Sum across source + target + action patterns (max 9)
   - Store result in `priority` column
2. Call ComputeSpecificity in store's CreateRule and UpdateRule for access rules
3. Extract access control logic from `policy.go` into `policy_access.go` as a standalone `http.Handler` middleware
4. Fix endpoint matching: replace direct string equality with `match.Match()` call
5. Update existing middleware tests, add specificity-order tests

### Phase 2: Token Bucket Rate Limiting
**Files**: `internal/middleware/policy_ratelimit.go`, `internal/middleware/policy_ratelimit_test.go`

1. Implement `RateLimitMiddleware` as `http.Handler` wrapper (standard nested wrapping per R2-d3)
2. Token bucket algorithm per rule:
   - State: `map[int64]*bucket` keyed by rule ID, protected by `sync.Mutex` (per R1-d3)
   - Bucket: `{tokens float64, lastRefill time.Time, capacity float64, refillRate float64}`
   - On request: refill tokens based on elapsed time, then consume 1 token
   - If no tokens: return 429 with `Retry-After` header (seconds until 1 token available)
3. Match incoming requests against rate_limit rules using same glob matching as access control
4. `PruneState(activeRuleIDs map[int64]bool)` method to remove orphaned buckets on snapshot (per R2-d4)
5. Unit tests: refill timing, exhaustion, Retry-After calculation, multiple rules, state pruning

### Phase 3: Circuit Breaker
**Files**: `internal/middleware/policy_circuit.go`, `internal/middleware/policy_circuit_test.go`

1. Implement `CircuitBreakerMiddleware` as `http.Handler` wrapper (standard nested wrapping per R2-d3)
2. State machine per route key `{source}.{target}.{action}`:
   - `map[string]*circuitState` keyed by route, protected by `sync.Mutex`
   - States: closed (pass-through, count failures) → open (reject all 503) → half-open (allow 1 probe)
   - Failure detection: HTTP 5xx responses only (per R2-d2)
   - Closed → Open: when failure count >= threshold within window
   - Open → Half-Open: when cooldown elapses
   - Half-Open → Closed: probe succeeds (non-5xx)
   - Half-Open → Open: probe fails
3. Response wrapping: lightweight `ResponseWriter` wrapper that captures only status code (not full body buffering)
4. Return 503 with `Retry-After: <cooldown_seconds>` when circuit is open
5. `PruneState(activeRouteKeys map[string]bool)` method to remove orphaned circuit states on snapshot (per R2-d4)
6. Unit tests: all state transitions, cooldown timing, concurrent access, state pruning

### Phase 4: Policy Snapshot Push
**Files**: `internal/policy/broadcaster.go`, `api/handlers_policy.go`, all middleware files

1. Modify broadcaster to accept `[]Rule` instead of single events — `Broadcast(rules []Rule)`
2. On any CRUD operation in handlers, query all enabled rules and broadcast full snapshot
3. SSE handler sends snapshot as single JSON event (type: "snapshot")
4. Update `policy.go` shared middleware infrastructure: remove `ApplyEvent` method, use only `UpdateRules` for snapshot application
5. Each stateful layer prunes orphaned state entries not in the new snapshot via `PruneState` (per R2-d4)
6. Update broadcaster tests for snapshot model

### Phase 5: Integration & Polish
1. Wire all three middleware layers in `policy.go` using nested wrapping: `accessMW(rateLimitMW(circuitBreakerMW(handler)))` (per R2-d3)
2. End-to-end test: full middleware chain with all three rule types active
3. Verify existing CRUD handler tests still pass
4. `gofumpt -w .` and `golangci-lint run`
5. `go test ./... -timeout 300s`

## 9. Contract Decisions

### Middleware Chain Order
```
Request → Access Control → Rate Limiting → Circuit Breaker → Handler
```
Access control first (cheapest check, rejects unauthorized callers before rate/circuit state is consulted).

### Rate Limit Response (429)
```json
{"error": "rate_limited", "rule_id": 5, "reason": "token bucket exhausted", "retry_after": 2}
```
Header: `Retry-After: 2`

### Circuit Breaker Response (503)
```json
{"error": "circuit_open", "rule_id": 8, "reason": "circuit breaker open for swarm-manager→my-svc", "retry_after": 30}
```
Header: `Retry-After: 30`

### Snapshot SSE Event Format
```json
{"type": "snapshot", "rules": [...all enabled rules...]}
```
Replaces the current incremental event types (created/updated/deleted).

### File Layout (per R2-d1)
```
internal/middleware/
  policy.go              — Shared types, config, rule cache, SSE subscription, NewPolicyMiddleware (wiring)
  policy_access.go       — Access control layer (specificity scoring, glob matching, 403 deny)
  policy_access_test.go  — Access control tests
  policy_ratelimit.go    — Token bucket rate limiting layer (per-rule state, 429 responses)
  policy_ratelimit_test.go — Rate limit tests
  policy_circuit.go      — Circuit breaker layer (3-state machine, 5xx detection, 503 responses)
  policy_circuit_test.go — Circuit breaker tests
  policy_test.go         — Integration tests for full chain + existing tests
```

## 10. Testing Plan

| Test Category | What | How |
|---|---|---|
| Specificity scoring | Score computation for various pattern combinations (exact, prefix glob, wildcard) | Table-driven unit tests for ComputeSpecificity |
| Endpoint matching fix | Glob patterns work in middleware (not just direct equality) | Unit tests mirroring evaluator's pattern tests |
| Rate limiter unit | Token bucket refill, exhaustion, Retry-After calculation, concurrent access | Time-controlled unit tests with synthetic clocks |
| Rate limiter state pruning | Orphaned buckets removed on snapshot update | Unit test: update rules, verify old state removed |
| Circuit breaker unit | State transitions: closed→open→half-open→closed, cooldown timing | Unit tests with synthetic failures and time control |
| Circuit breaker failure detection | Only 5xx counts as failure, 4xx does not trip circuit | Explicit test cases for each status code category |
| Circuit breaker state pruning | Orphaned circuit states removed on snapshot update | Unit test: update rules, verify old state removed |
| Middleware chain | Three layers compose correctly, correct rejection order | httptest with real middleware chain |
| Snapshot push | Rule change triggers full snapshot, receivers update atomically | Integration test with broadcaster + middleware |
| Existing CRUD | All existing handler tests continue to pass | Run existing test suite |

## 11. Rollout / Validation Checklist
- [ ] `go test ./... -timeout 300s` passes all tests (existing + new)
- [ ] `gofumpt -l .` reports no formatting issues
- [ ] `golangci-lint run` passes
- [ ] Access control specificity scoring works end-to-end
- [ ] Endpoint matching uses glob patterns (bug fixed)
- [ ] Rate limiting returns 429 with correct Retry-After
- [ ] Circuit breaker transitions through all three states correctly
- [ ] Circuit breaker only counts 5xx as failures (not 4xx)
- [ ] Policy snapshot push replaces cached rules atomically
- [ ] Stateful layers prune orphaned state on snapshot
- [ ] Existing CRUD API and SSE tests still pass

## 12. Risks + Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| In-memory rate limit state lost on restart | Brief burst window after restart | Acceptable: bucket refills naturally, no persistent damage |
| Circuit breaker false opens under bursty errors | Legitimate requests rejected | Configurable threshold + cooldown; half-open probing allows recovery |
| Snapshot push bandwidth with many rules | SSE message size grows linearly | Acceptable at current scale (dozens of rules). Monitor if rule count grows significantly. |
| Middleware ordering matters | Wrong order gives confusing errors | Document and enforce access → rate limit → circuit breaker order |
| Response wrapping for circuit breaker | Must observe status code without buffering entire response | Use lightweight ResponseWriter wrapper that captures only status code |
| Stale circuit breaker state after snapshot prune | Clean state means circuit resets to closed | Acceptable: conservative default, prevents false opens from old state |
| Extracting access layer from policy.go | Could break existing tests if refactor is incomplete | Phase 1 ends with all existing 18 tests passing against new file structure |

## 13. Non-goals / Prohibited Patterns
- Do NOT add distributed rate limiting or cross-process state sharing
- Do NOT modify the discovery package
- Do NOT add authentication to policy endpoints
- Do NOT create a `lib/` directory
- Do NOT mock SQLite in tests
- Do NOT change the HTTP router
- Do NOT modify the server-side evaluator (internal/policy/evaluator.go) — it stays access-only
- Do NOT keep access, rate limit, and circuit breaker logic in a single file — use separate files per R2-d1

## 14. Definition of Done
- [ ] Access control with auto-computed specificity scoring (exact=3, prefix=2, wildcard=1, max 9)
- [ ] Endpoint matching bug fixed (glob patterns in middleware)
- [ ] Token bucket rate limiting middleware with per-rule in-memory state (sync.Mutex)
- [ ] Circuit breaker middleware with 3-state machine (5xx-only failure detection)
- [ ] Policy snapshot push via SSE on any rule change
- [ ] Three-layer middleware chain as nested http.Handler wrapping
- [ ] Separate files per middleware layer (policy_access.go, policy_ratelimit.go, policy_circuit.go)
- [ ] Stateful layers prune orphaned state on snapshot update
- [ ] All existing tests passing
- [ ] New tests for each middleware layer + integration tests
- [ ] Code formatted with gofumpt and passing golangci-lint
