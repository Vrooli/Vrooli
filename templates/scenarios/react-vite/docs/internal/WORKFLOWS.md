# Workflows — {{SCENARIO_DISPLAY_NAME}}

Temporal workflows make lifecycle behavior explicit. Use them when a
domain or UI feature has named states and only some events are valid in
each state.

## When to model a workflow

Add workflow code when correctness depends on:

- ordered state changes,
- retries, cancellation, stale completion, or double-submit behavior,
- async completion from background work,
- locks, leases, schedules, polling, or concurrency limits,
- UI modes where only one mode can be active at a time.

Do not add workflow files for plain CRUD, generated adapters, static
configuration, formatting helpers, or presentational UI with no illegal
transitions.

## Maturity ladder

Temporal workflows mature in layers. Do not skip the executable layers
to add a standalone formal document.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, components, callbacks, or jobs. |
| 1 | Inventory | The flow is listed in `docs/internal/TEMPORAL-FLOWS.md` with source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative spec | A domain-local `*.spec.json` declares states, events, transitions, invariants, traces, and formal-model status; tests fail if it drifts from the matrix/traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool checks the model and generates deterministic artifacts replayed by production tests. |

The template ships Level 4 examples for the notes attachment upload
workflow:

- API: `api/internal/notes/attachment_upload_workflow.spec.json`
- UI: `ui/src/features/notes/AttachmentUploadWorkflow.spec.json`

## Production shape

API domains that own durable lifecycle state use:

```
api/internal/<domain>/
  <flow>_workflow.go
  <flow>_workflow.spec.json
  <flow>_workflow_test.go
```

UI features that own client-side modes use:

```
ui/src/features/<domain>/
  <domain>Workflow.ts
  <domain>Workflow.spec.json
  <domain>Workflow.test.ts
```

The workflow owns state/status values, events, `Transition`, and
`CheckInvariants`. It should be pure or nearly pure. Effects live
outside the workflow behind seams: repositories, BlobStore, clocks,
timers, HTTP clients, or UI API modules.

## Matrix and trace tests

Every modeled workflow should have a transition matrix that covers all
state/event pairs and representative traces that replay realistic
flows. The test utilities under `api/internal/testutil/modeltest/` and
`ui/src/test-utils/modeltest/` enforce:

- no duplicate state/event rows,
- no unknown states or events,
- no missing state/event pairs,
- explicit expected transitions or explicit expected errors,
- trace replay against the production transition function.

This is stricter than line coverage. Coverage can prove a branch ran;
matrix completeness proves every production state/event pair was
classified.

## Declarative specs

A `*.spec.json` file is the intended transition contract. It is not a
replacement for production transition code or executable tests. The
spec must stay beside the owning workflow so deleting the domain removes
the model, tests, and spec together.

Include:

- stable flow id and domain,
- state and event lists,
- initial state and terminal states,
- one transition row per state/event pair,
- invariants stated in domain language,
- representative traces,
- formal-model status (`not_adopted`, `candidate`, or `adopted`).

The test utilities under `api/internal/testutil/modeltest/` and
`ui/src/test-utils/modeltest/` include spec-conformance helpers. Use
them so drift between the declarative spec and hand-authored matrix or
trace tests fails locally.

## Formal spec roadmap

Quint/TLA+ can be added later for complex state spaces, but not as
standalone documentation. A formal model is considered adopted only
when:

1. the model is checked by its toolchain,
2. deterministic traces or matrices are generated from the model,
3. production Go/TypeScript transitions replay those artifacts in
   tests,
4. validation fails when artifacts are stale or implementation behavior
   diverges.

Until that loop exists, the production workflow plus matrix/trace tests
is the source of truth.
