---
name: "resilience-lenses"
description: "Six runtime-behavior audit lenses for a scenario: temporal flow, idempotency, failure topography, error semantics, signal feedback, progress continuity. Pick the lens the finding belongs to."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","architecture"]
  tags: ["skill", "resilience"]
  icon: "box"
  status: "active"
  targetDimensions: ["cycles","structure"]
  defaultScope: "work-scope"
  revision: 1
  createdAt: "2026-11-20T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Steer focus: Resilience Lenses

**Scope mode: `architecture`.** Hold to that row of `work-scope` for this session.

One skill for how `scenarios/{{TARGET}}/` behaves over time and under stress. Each lens is a question; pick the lens a finding belongs to and do not patch one lens's problem with another's change. Read and update scenario docs through `knowledge-observatory-tools`: `docs/internal/TEMPORAL-FLOWS.md`, `INVARIANTS.md`, `SEAMS.md`, the `error-semantics` doc, `PROBLEMS.md`. Record findings in those existing docs; no standalone audit files. Structural findings (wrong folder, hidden seam, unnamed concept) go to `architecture-lenses`. Do not break behavior, regress tests, weaken a safety guarantee, or add features or a new observability stack or global error framework; every change keeps or improves completeness and reliability. Validation scope follows `docs/TESTING.md`.

| Finding | Lens |
|---|---|
| Named lifecycle states, illegal transitions, stale completion, double submit, out-of-order events, leases, polling | Temporal flow |
| Running it twice is worse than once; retries duplicate work or effects | Idempotency |
| Unknown or unmapped ways the scenario fails; dependency outages; misleading success | Failure topography |
| Errors are ad-hoc strings or shapes; recovery path unclear | Error semantics |
| Important states, transitions or failures are silent, fragmented or misleading | Signals |
| Work cannot be safely stopped and resumed | Progress continuity |

---

### 1. Temporal flow

Make time-dependent behavior explicit, testable and low-drift. A flow needs a workflow model when it has named lifecycle states over time, some transitions are illegal, retries, cancellation, stale completion, double submit or out-of-order completion can corrupt state, the UI can show contradictory states (loading and success together), or correctness depends on leases, locks, schedules, polling or concurrency limits. Typical: uploads and imports, jobs and queues, retries, approvals, auth and billing lifecycles, one-active-mode UI, multi-step orchestration. Plain CRUD, pure mapping, thin handlers, and presentational UI do not; use ordinary validation tests there.

Keep the model beside the owning capability (`api/internal/<domain>/<flow>_workflow.go` with `_workflow.spec.json` and `_workflow_test.go`; `ui/src/features/<domain>/<Flow>Workflow.ts` likewise). If deleting the capability should delete the workflow, it lives there; helpers for many domains go in `internal/testutil` or the UI test utilities. The workflow is pure or nearly pure and owns state and event values, `Transition(state, event) -> next state or typed error`, `CheckInvariants(state)`, and terminal-state rules. Database writes, network, filesystem, clocks, sleeps, timers, goroutines and React effects stay outside; services, handlers, jobs and components orchestrate effects around it.

Ladder: 0 implicit in handlers, callbacks, sleeps or shared mutable state; 1 inventory row in `TEMPORAL-FLOWS.md` (flow, risk, source links, maturity, next step); 2 workflow model beside the domain; 3 matrix and traces (every state and event pair has exactly one expected result, duplicates, unknowns and missing pairs fail, traces replay against the production transition function, terminal states cannot be escaped, stale completion, retry, cancel, reset, duplicate submit and out-of-order events covered where relevant); 4 a domain-local declarative spec (flow id, states, events, initial and terminal states, one row per pair, invariants in domain language, traces, formal-model status `not_adopted|candidate|adopted`) that tests compare against the matrix so drift fails; 5 a checked Quint or TLA+ model that generates matrices and traces production tests replay. Stop where risk and time justify; most production workflows reach 3, important ones 4. Level 5 is adopted only when the toolchain checks the model, artifacts are generated from it, production code replays them, and stale artifacts fail validation; otherwise record the target in `TEMPORAL-FLOWS.md`. Do not call the spec the source of truth until tests consume it. Coverage percentages are not proof: a suite can hit every line and never check a forbidden transition.

### 2. Idempotency and replay safety

Goal: running something twice is no worse than running it once, and replays (intentional or accidental) give stable, correct results. Read `INVARIANTS.md` Replay/Idempotency section first. Find paths that write, call out, enqueue, trigger downstream effects, or depend on time, randomness or global state, and check whether repetition duplicates or corrupts state, whether a natural idempotency key or replay guard exists, and whether "already done" is distinguishable from "not yet done" (never applied, applied once, applied already and a no-op). Set commit boundaries so a partial attempt never commits incomplete state; prefer atomic or transaction-like behavior. Make results stable for identical inputs: wrap current time, seed or scope randomness, avoid counters that do not survive retries. Prevent double application with idempotency keys, stable request ids, memoization of expensive deterministic work, and "already completed" short-circuits, without removing legitimate side effects and without adding flags that do not improve correctness. Resuming an interrupted multi-step process must pick up exactly where it should, skip no required step, repeat no irreversible mutation, and state the current position instead of guessing. Add tests that run the action twice and assert the same end state.

### 3. Failure topography and graceful degradation

Map how the scenario fails, then make failure safe, observable and recoverable. Read the `error-semantics` doc, list the critical flows (journeys, background jobs, integrations) with inputs, steps, dependencies and outputs, and enumerate realistic failure modes (service timeouts or errors, invalid or missing input, transient network, inconsistent data, races and ordering, resource exhaustion and rate limits) classified by scope (local or systemic), frequency and impact. Skip contrived failures. For high-impact ones trace what happens in the UI, the logic and the state, and what is logged: silent or swallowed errors, partial updates that leave a confusing state, misleading success surfaces, and harsh failures where a softer fallback exists. Then degrade gracefully on the important paths: keep core function when a dependency is down, fall back where a safe fallback exists, tell the user what is unavailable and what still works, never fake success, and never lose data to reach a fallback. Add a test per degraded path that induces the dependency failure.

### 4. Error semantics and recovery paths

Turn "something went wrong" into a few categories with explicit recovery. Derive error domains across UI, jobs, API in and out, and integrations, and map every important failure path to a category rather than an ad-hoc message or raw exception. Good categories are mutually exclusive, recovery-distinct (two categories with the same recovery are one), user-recognizable ("validation", not "schema mismatch"), four to eight in number, and stable across the codebase. Normalize representation incrementally (named types or codes, tagged results, one payload shape per surface) without a new global framework, and never leak stack traces, internal ids or secrets to clients.

Recovery per category: user can fix the cause, so show what is wrong and keep their input; transient, so retry with backoff or offer retry; partial progress is valuable, so preserve it and continue from a checkpoint; user needs to know, so message with next steps, otherwise log only; an agent can handle it, so return machine-readable category, severity and hint rather than free text. Anti-patterns: "retry" for validation errors, technical detail for transient failures, silent failures, generic "contact support". Messages are clear, non-technical, non-blaming and say what to do next; status codes, UI text and internal errors tell one coherent story. Tie errors to diagnosis: log category and correlation id once at the boundary. Audit with `rg` for error origins and catches, generic messages, empty catch blocks, and error UIs without an action. Preserve behavior while normalizing; document categories in the `error-semantics` doc.

### 5. Signals and feedback

Make the scenario self-explanatory at runtime for humans and agents. From the PRD and main workflows, name the minimum that must be visible: key states (idle, in progress, degraded, blocked), transitions (start and finish, retry, fallback, cancel), and high-impact failures (data-loss risk, dependency problems, invariant violations), to users where relevant and to operators and future agents always. Audit the logs, status commands, state fields, UI feedback and tests for events that happen silently, ambiguous or too low-level signals, noisy logs, and signals fragmented across places. Use the scenario's domain language, not implementation jargon; give each major workflow a small coherent set of signals (what was attempted, what happened, success or failure, next step). Users get timely progress, completion and actionable failure messages, avoiding both silence and technical noise. Do not expose internals that do not help a decision, and do not add a new observability stack.

### 6. Progress continuity and interruption resilience

Make work safely stoppable and resumable for users and agents. Read `TEMPORAL-FLOWS.md` Checkpoint Flows first. Identify multi-step creation or configuration, long-running jobs, and agent workflows; for each define what progress means, the natural checkpoints, and what completion looks like. Separate ephemeral UI state (focus, scroll, open panel) from meaningful progress state (steps done, decisions made, data entered), and represent the latter explicitly (flags, phases, stored inspectable state, an identifiable current position); never infer a step from "this field is non-empty". Persist at safe boundaries (after a meaningful step, a confirmed choice, a finished or failed long operation) with checkpoints that are idempotent and carry enough context to resume without guessing; if full persistence is too risky this loop, make the checkpoint concept explicit and prepare the structure. Give resume an entry point that finds the current state, shows the next step or an explicit choice, and recaps context, never dropping the user into ambiguity about what is done, what needs redoing, and what is safe to retry.

---

### Output

Finish with at least one lens advanced a level or one concrete finding fixed with a test, behavior preserved, the change small and incremental, and findings filed in the existing docs. Record substantial redesign in `PROBLEMS.md` rather than half-doing it.
