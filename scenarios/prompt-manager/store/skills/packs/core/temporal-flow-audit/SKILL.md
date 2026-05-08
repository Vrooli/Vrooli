## Steer focus: Temporal Flow Audit

Prioritize **making time-dependent behavior explicit, testable, and low-drift** in `scenarios/{{TARGET}}/`. Classify temporal risk first, then move qualifying flows toward domain-owned workflow models, matrix/trace conformance, declarative specs, and eventually checked formal models.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read screaming-architecture-audit` — use when temporal logic is buried outside the domain that owns the capability.

Read first when present:
- `scenarios/{{TARGET}}/docs/internal/TEMPORAL-FLOWS.md` — prior temporal-flow inventory and maturity status.
- `scenarios/{{TARGET}}/docs/internal/WORKFLOWS.md` — scenario-local workflow conventions, if the scenario has them.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — effect boundaries that should stay outside pure transition logic.

---

### 1. Scope Boundaries

**In scope:**
- async operations, background jobs, retries, polling, scheduled work, streaming, cancellation, stale completion, leases, locks, lifecycle modes, and multi-step orchestration
- extracting or improving domain-owned workflow logic for flows with named states/events and illegal transitions
- adding or improving matrix, trace, concurrency, cancellation, stale-artifact, and spec-conformance tests
- updating `docs/internal/TEMPORAL-FLOWS.md` as a navigation and continuity index

**Out of scope:**
- modeling plain CRUD with no lifecycle constraints
- adding formal specs as standalone documentation
- broad rewrites of unrelated architecture, UI, storage, or APIs
- duplicating full transition tables in docs when a domain workflow/spec file is the better source of truth

---

### 2. Temporal Maturity Model

Assess each candidate flow against this ladder. Move only as far as the current risk and available time justify.

| Level | Name | What exists | When to stop here |
|---|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior is implicit in handlers, components, callbacks, jobs, sleeps, or shared mutable state. | Do not stop here for high-risk flows; record as a candidate if not fixed now. |
| 1 | Inventory | `TEMPORAL-FLOWS.md` lists the flow, risk, source links, current maturity, and next step. | Low-risk async behavior where discovery is the immediate goal. |
| 2 | Workflow model | State/status values, event/command values, `Transition`, and `CheckInvariants` live beside the owning domain/feature. | Small flows where state/event surface is obvious and already covered by focused tests. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay realistic traces against production transition logic. | Most production workflows should reach at least this level. |
| 4 | Declarative spec | A domain-local `*.spec.json` or equivalent declares states, events, transitions, invariants, terminal states, traces, and formal-model status; tests fail if it drifts from matrix/traces. | Current preferred target for important but not yet formally checked workflows. |
| 5 | Checked formal model | Quint/TLA+ or equivalent is checked by its toolchain and generates deterministic matrices/traces replayed by production tests. | Large, concurrent, or business-critical state spaces where hand-authored matrices are not enough. |

The maturity level is not a vanity score. It tells future agents what confidence exists and what kind of drift is still possible.

---

### 3. Decide What Needs Temporal Modeling

Not every async path needs a workflow model. Use this table before adding workflow-shaped files:

| Question | If YES | If NO |
|---|---|---|
| Can the thing be in named lifecycle states over time? | Consider a workflow model. | Prefer ordinary validation/table tests. |
| Are some state changes illegal? | Extract explicit state/event/transition rules. | Avoid state-machine ceremony. |
| Can retries, cancellation, stale completion, double-submit, or out-of-order completion corrupt state? | Add matrix, trace, and race/cancellation tests. | Happy/error-path tests may be enough. |
| Can the UI represent contradictory states such as loading and success together? | Use a discriminated union/reducer-style workflow. | Local component state may be fine. |
| Does correctness depend on leases, locks, schedules, polling, or concurrency limits? | Treat it as temporal logic. | Keep the code simple. |

Typical flows that **do** deserve temporal audit attention:
- uploads and imports
- jobs, tasks, runs, queues, and background work
- retries, polling, scheduled work, leases, locks, approvals
- auth/session lifecycle, billing/subscription lifecycle
- UI modes where only one mode may be active at a time
- multi-step orchestration across resources or scenarios

Typical flows that **do not** need workflow modeling:
- plain CRUD where create/read/update/delete has no lifecycle constraints
- pure formatting, mapping, generated-code adapters, and static config
- thin handlers/commands that only translate into one service call
- presentational UI with no meaningful ordering or impossible-state risk

---

### 4. Canonical Workflow Shape

When a flow qualifies as temporal logic, keep the model beside the capability that owns it. Use `screaming-architecture-audit` if the flow is spread across generic services, shared utilities, or UI components with no clear domain owner.

Preferred API/domain shape:

```text
api/internal/<domain>/
  <flow>_workflow.go
  <flow>_workflow.spec.json      # Level 4 when adopted
  <flow>_workflow_test.go
```

Preferred UI/domain shape:

```text
ui/src/features/<domain>/
  <Flow>Workflow.ts
  <Flow>Workflow.spec.json       # Level 4 when adopted
  <Flow>Workflow.test.ts
```

Equivalent shapes are fine in non-Go/React stacks. Preserve the same ownership rule: if deleting the capability should delete the workflow, keep the workflow in that capability's domain. Shared helpers for many domains belong in `internal/testutil`, `ui/src/test-utils`, or the scenario's equivalent test-support package.

The workflow owns:
- state/status values
- event/command values
- `Transition(state, event) -> next state or typed error`
- `CheckInvariants(state) -> typed error/nil`
- terminal-state rules

The workflow should be pure or nearly pure. Keep database writes, network calls, filesystem access, clocks, sleeps, timers, process globals, React effects, and goroutines outside it. Services, handlers, jobs, and components orchestrate side effects around the transition instead of duplicating transition rules.

---

### 5. Matrix, Trace, and Spec Conformance

For Level 3+ workflows, prove temporal completeness with tests that fail on drift:
- every production state/status is represented
- every production event/command is represented
- every state/event pair has exactly one expected result
- duplicate, unknown, or missing pairs fail the test
- traces replay step-by-step against the production transition function
- terminal states cannot be escaped unless the product explicitly allows it
- stale completion, retry, cancel, reset, duplicate submit, and out-of-order events are covered where relevant

Coverage percentages are not proof of temporal completeness. A suite can hit every line while never checking a forbidden transition.

For Level 4, add a domain-local declarative spec. Include:
- stable flow id and domain
- states and events
- initial state and terminal states
- one transition row per state/event pair
- invariants in domain language
- representative traces
- formal-model status (`not_adopted`, `candidate`, or `adopted`)

Do not claim the spec is the sole source of truth unless tests consume it. Until then, phrase it as the intended transition contract. Preferred current setup: tests compare the spec against the executable matrix/traces so drift fails locally.

---

### 6. Formal Specs

Quint/TLA+ models are useful when the state space is large enough that hand-authored matrices are no longer enough. Do not add a formal spec as documentation only.

A Level 5 formal model is adopted only when:
1. the model is checked by its toolchain,
2. deterministic traces, transition matrices, or equivalent artifacts are generated from the model,
3. production Go/TypeScript transition functions replay those artifacts in tests,
4. validation fails when artifacts are stale or production behavior diverges.

If that loop cannot be added in the current pass, record the formal-model target in `scenarios/{{TARGET}}/docs/internal/TEMPORAL-FLOWS.md` and keep the production workflow plus matrix/trace/spec-conformance tests as the current source of confidence.

---

### 7. Audit Checklist

Map temporal candidates:
- requests and long-running operations
- loading and initialization sequences
- background jobs, queues, polling loops, scheduled work
- streaming, subscriptions, timers, listeners, and teardown paths
- retries, cancellation, stale completion, double-submit, and concurrency limits

For each candidate, identify:
- trigger
- owner domain/feature
- states and events, if any
- side effects and seams
- completion and failure signals
- race risks and ordering assumptions
- current maturity level
- next maturity step

Then improve the highest-risk flows first:
- replace arbitrary sleeps/delays with event- or state-based coordination
- make parallel vs sequential operations explicit
- add idempotency or conflict guards for repeated actions
- ensure teardown stops timers, listeners, subscriptions, and background work
- distinguish transient vs persistent failures
- make half-updated states visible and recoverable

Avoid large risky rewrites in one loop. If the correct redesign is too broad, document the candidate and the next concrete step.

---

### 8. Documentation

Use `knowledge-observatory-tools` to read and update `scenarios/{{TARGET}}/docs/internal/TEMPORAL-FLOWS.md`.

This doc is an index and memory layer, not the detailed transition source of truth. Detailed states/events/transitions/invariants belong in domain workflow/spec files and tests.

Recommended shape:

```markdown
# Temporal Flows

## Flow Index

| Flow ID | Domain | Risk | Model Status | Source of Truth | Tests | Remaining Gaps |
|---|---|---|---|---|---|---|

## Unmodeled Candidates

| Candidate | Why It May Be Temporal | Current Risk | Recommended Next Step |
|---|---|---|---|

## Declarative & Formal Spec Status

| Flow ID | Spec Type | Generated Artifacts | Drift Check | Status |
|---|---|---|---|---|

## Audit Notes

- [Date] [Agent/author]: [Short note with evidence and links.]
```

When updating the doc:
- verify existing claims against code before extending them
- link to source files and tests with `path:` references
- record unmodeled candidates instead of leaving discoveries in chat
- keep long transition tables out of the doc once a domain spec/test exists
- create `path:scenarios/{{TARGET}}/docs/internal/` if needed

---

### 9. Output Expectations

By the end of this loop, the scenario should:
- have a clearer inventory of time-dependent flows
- have fewer hidden ordering and concurrency assumptions
- move high-risk flows up the maturity ladder
- keep temporal rules in the owning domain rather than scattered across handlers/components
- have executable validation for important state transitions
- leave future agents with a lower-drift `TEMPORAL-FLOWS.md` index

Avoid superficial edits that rename or reshuffle code without improving temporal confidence.
