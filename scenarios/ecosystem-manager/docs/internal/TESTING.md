# Testing — Ecosystem Manager

How to write tests against this scenario's shape. Read this *before*
your first non-trivial test — the patterns below are load-bearing for
the gates run by `vrooli scenario test ecosystem-manager`.

Ecosystem-manager is a REST/JSON Go API ([CODE: api/pkg/server/server.go],
`gorilla/mux`) whose core domain is the **auto-steer** control loop. The
defining test pattern here is **control-loop testing**: substitute the
completeness `Provider` and the test-genie `AuditRunner` sensors with
synthetic findings/score and drive the loop's decisions deterministically —
no real agents, no real measurement.

> **Test command:** `vrooli scenario test ecosystem-manager` (or the
> scenario Makefile target `make test`). For a fast inner loop on the
> Go core: `cd scenarios/ecosystem-manager/api && go test ./... -timeout 300s`.

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **Control-loop orchestration**: [CODE: api/pkg/autosteer/execution_orchestrator_test.go]
  — constructs the orchestrator with `NewMockCompletenessProvider()`, a fake
  findings `AuditRunner`, and an in-memory `MockExecutionStateRepository`,
  then asserts the diagnose→select→execute→measure mini-loop shrinks findings
  and terminates, plus the budget-cap halt.
- **Greedy selection**: [CODE: api/pkg/autosteer/selector_test.go]
  — pins that `SelectNextSkill` picks the heaviest weighted dimension with an
  eligible skill, respects dimension weights, and falls through / selects
  nothing when no skill is eligible.
- **Termination gates**: [CODE: api/pkg/autosteer/terminator_test.go]
  — drives `ShouldStop` through objective-met, operational-targets gate,
  diminishing-returns, and budget-exhausted with synthetic state.
- **HTTP handlers**: [CODE: api/pkg/autosteer/handlers_test.go] (profile CRUD)
  and [CODE: api/pkg/autosteer/handlers_error_shape_test.go] (coverage,
  structured-error, not-found branches) — mount handlers and assert status
  code + JSON body.
- **Queue integration**: [CODE: api/pkg/queue/autosteer_integration_test.go]
  — seeds task shapes and asserts `ShouldContinueTask` (recycle vs stop).

If your test doesn't look like one of these, ask why before shipping.

## API testing

### Layout

```
api/pkg/
├── autosteer/                    # the control loop + its handlers + tests
│   ├── repositories.go           # ProfileRepository, ExecutionStateRepository,
│   │                             #   ExecutionHistoryRepository
│   ├── interfaces.go             # PromptEnhancerAPI
│   ├── repositories_mock.go      # MockCompletenessProvider, MockExecutionStateRepository, …
│   ├── selector.go / .._test.go              # greedy skill selection
│   ├── terminator.go / .._test.go            # global termination gates
│   ├── eligibility.go / .._test.go           # skill eligibility
│   ├── execution_orchestrator.go / .._test.go
│   ├── decision_trace.go / .._test.go
│   └── handlers.go / handlers_test.go / handlers_error_shape_test.go
├── completeness/                 # the measurement sensor (Connect client)
│   └── score.go                  # Provider (Score), Client
├── findings/                     # the diagnose sensor
│   └── runner.go                 # AuditRunner, TestGenieRunner, FakeRunner
├── agentmanager/                 # outbound run start/stop (the actuator)
│   ├── api.go                    # AgentServiceAPI
│   └── mock.go                   # MockAgentService
├── queue/                        # task queue + auto-steer recycle decision
│   └── autosteer_integration.go / .._test.go
└── server/server.go              # gorilla/mux router (REST/JSON)
```

### Handler test pattern (REST/JSON)

Handlers are plain `http.HandlerFunc`s registered on a `mux.Router`. A
handler test constructs the handler with fake services, issues a request,
and asserts the status code and decoded JSON body. Success goes through
`writeJSON`; failures go through `writeError` and decode into the
`ErrorResponse` envelope (`{error, message, code}` — see
[ERROR-HANDLING.md](ERROR-HANDLING.md)).

The reference cases live in [CODE: api/pkg/autosteer/handlers_test.go]:
profile CRUD (201/200/404/400), and
[CODE: api/pkg/autosteer/handlers_error_shape_test.go]: coverage reports,
structured-error envelopes, and the not-found execution-state branch.

## Auto-steer control-loop testing

This is the scenario's signature pattern and the reason the seams exist.

**Mental model.** The auto-steer loop is a closed-loop controller
(see [../concepts/CONTROL-MODEL.md](../concepts/CONTROL-MODEL.md)):

```
completeness Score + test-genie findings (sensors) ─▶ Selector + Terminator (control law)
        ▲                              │
        │                              ▼
 ExecutionStateRepository  ◀── ExecutionOrchestrator ──▶ agent-manager (actuator)
        (controller state)
```

**The pattern.** Substitute the completeness `Provider` and the findings
`AuditRunner` with fakes that return a synthetic `Score` / findings vector.
Because the sensors now return exactly the values you choose, every
downstream decision becomes deterministic and table-driven — **without
running a real agent or collecting real measurements**:

- **Greedy selection** — set the findings so a given dimension is heaviest
  (profile weight × open severity) and assert `SelectNextSkill` picks the
  first eligible skill for it, or selects nothing when none is eligible
  ([CODE: api/pkg/autosteer/selector.go]).
- **Termination** — drive `ShouldStop` through each reason: objective-met
  (no finding above `max_open_severity`, op-targets gated), diminishing
  returns (trailing improvement below floor), and budget exhausted
  ([CODE: api/pkg/autosteer/terminator.go]).
- **Mini-loop convergence** — wire the orchestrator with shrinking synthetic
  findings and assert it terminates (and that the budget cap halts a
  non-converging loop) ([CODE: api/pkg/autosteer/execution_orchestrator.go]).
- **Requeue-vs-stop** — seed a `TaskItem` and assert `ShouldContinueTask`
  returns continue / stop-when-complete
  ([CODE: api/pkg/queue/autosteer_integration.go (180-227)]).

**Why fake the sensors instead of the whole loop.** Real measurement (the
completeness Connect client over scenario-completeness-scoring) and real
test-genie audits shell out to other scenarios — slow, non-deterministic,
and side-effecting. Faking at the `Provider` / `AuditRunner` seams keeps the
*real* control law (selector + terminator + orchestrator) under test while
removing the only non-deterministic inputs.

## UI testing

The UI is React + Vite (every Vrooli scenario ships a UI). UI tests use
Vitest with `vi.mock` hoisting to substitute the per-domain API client
modules, render through a providers wrapper, and assert against test IDs
and translation keys rather than brittle copy. The auto-steer execution
hook treats a `404` from `/auto-steer/execution/{taskId}` as "no state
yet" (undefined) while surfacing other statuses as real errors — mirror
that contract in hook tests by stubbing the client to reject with a
404-carrying `ApiError` and asserting the hook returns undefined, not an
error state. Run UI tests through `vrooli scenario test ecosystem-manager`,
which counts eslint warnings as failures — drive UI lint to zero.

## CLI testing

The CLI wraps the same REST endpoints. CLI command tests spin a real
`httptest.Server` exposing the relevant handlers, point the CLI client at
it, and capture stdout to assert rendered output. Keep CLI tests at the
command-dispatch + rendering layer; the business logic is already covered
by the autosteer unit tests, so CLI tests should prove argument parsing,
endpoint selection, and human-readable formatting — not re-test the loop.

## Coverage thresholds

Coverage is enforced by `vrooli scenario test ecosystem-manager` /
`make test`; do not chase a percentage as proof of correctness. A suite
can touch every line of the control loop while never testing "stop on
diminishing returns" or "select nothing when no skill is eligible." Prefer
the branch-shaped control-loop cases above (each stop reason, each
requeue-vs-stop outcome) over line-coverage padding. The auto-steer
package carries the meaningful coverage; handlers need the 4xx/5xx
branches exercised, not just the happy path.

## Common patterns and anti-patterns

**Do:**

- Fake the completeness `Provider` with `MockCompletenessProvider` (and the
  findings `AuditRunner` with a fake) for any test that touches loop
  decisions — it is the cheapest path to determinism.
- Use the in-memory `MockExecutionStateRepository` for orchestrator
  tests so iteration state and the decision trace are fully controllable.
- Use `MockAgentService` ([CODE: api/pkg/agentmanager/mock.go]) to assert
  the actuator was invoked (`StopRun` called with run X) and to exercise
  upstream-failure branches (agent-manager unreachable, StopRun error).
- Decode handler error bodies into `ErrorResponse` and assert
  `code`/`message`, not on raw strings.

**Don't:**

- Don't run a real agent, a real completeness RPC, or a real test-genie
  audit in a unit test — that reintroduces the non-determinism the sensor
  seams remove.
- Don't assert loop behavior through the HTTP layer when a unit test on
  the orchestrator/coordinator is more direct.
- Don't seed a real filesystem queue (`queue/<status>/`) or a real
  SQLite file when the in-memory storage fake and
  `MockExecutionStateRepository` express the same precondition.
- Don't treat coverage percentage as completeness — enumerate stop
  reasons and requeue outcomes explicitly.

## Cross-references

- [SEAMS.md](SEAMS.md) — the seams these tests substitute (completeness `Provider`, `AuditRunner`, `ExecutionStateRepository`, `AgentServiceAPI`, …)
- [../concepts/CONTROL-MODEL.md](../concepts/CONTROL-MODEL.md) — auto-steer as a closed-loop controller
- [ERROR-HANDLING.md](ERROR-HANDLING.md) — `ErrorResponse` envelope asserted in handler tests
- [../concepts/ARCHITECTURE.md](../concepts/ARCHITECTURE.md) — overall scenario architecture
