# Flows — Unit Health

This document is the canonical workflow and state-transition map for
the scenario. Use it when behavior depends on ordered states, retries,
cancellation, stale completion, background jobs, polling, or mutually
exclusive UI modes.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

Plain CRUD with no meaningful ordering constraints does not need a
workflow model.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Scenario validation run | validation | `ValidateScenario` from UI, CLI, or Test Genie. | Normalized findings, maturity assessment, and (when executed) persisted run history. | Ordered pipeline with degraded and failure paths; each executed command has a bounded lifecycle. | Level 1 (inventory) for the pipeline; executor and service tests cover the ordering and bounds directly. |

## Flow Details

### Scenario validation run

- Owner domain: validation (`api/internal/validation/service.go`).
- Trigger: `ValidationService.ValidateScenario` or the shared
  `ScenarioValidationService.ValidateScenario`.
- Inputs: scenario slug or path, optional workspace ids,
  `include_execution`, `use_cache`, `fast_test_only`.
- Steps:
  1. Locate the target (`discovery.Locator`) and resolve its root.
  2. Discover surfaces and parse units through Code Facts
     (`discovery.Discoverer`). If Code Facts is unreachable, build a
     heuristic fallback inventory and set `degraded_reason`.
  3. Build the per-workspace plan: canonical framework, fast and
     coverage commands, hermetic policy, resource limits.
  4. Optionally execute planned commands through `executor.Runner`
     under a per-command timeout, a no-output watchdog, process-group
     cleanup, and weighted admission caps. Reuse cached evidence when
     the evidence key matches.
  5. Run the analyzers (coverage, architecture, quality, reliability,
     diagnostics, requirement traceability, policy projections).
  6. Assess maturity against the `maturity` block of
     `.vrooli/test-genie.json` and roll findings into `status`.
  7. Persist the run through `runhistory.Store` when execution ran and
     a store is configured.
- Outputs: `ValidateScenarioResponse` with `status` of `passed`,
  `failed`, `degraded`, or `error`.
- Failure modes: unresolvable target (`invalid_argument`), Code Facts
  unavailable (degraded), missing toolchain, test failure, timeout,
  no-output stall, system error, unsupported command.
- Retry/cancel behavior: the request context cancels execution;
  process groups are torn down on cancel or timeout. Callers may
  simply re-run; the evidence cache makes an unchanged target cheap.
- Tests: `api/internal/validation/service_test.go`,
  `execution_test.go`, `execution_evidence_test.go`, `e2e_test.go`,
  `api/internal/executor/*_test.go`, `api/handlers/validation/handler_test.go`,
  `ui/src/features/validation/ScenarioValidationWorkbench.test.tsx`.
- Requirements: see `requirements/`.

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| validation / executed command | running → `passed`, `failed`, `timeout`, `error` (terminal), each with a failure class (`test_failure`, `missing_dependency`, `misconfiguration`, `timeout_hang`, `no_output_stall`, `system`, `unsupported`) | Leaving a terminal state; reporting output after process-group teardown | `api/internal/executor` (timeout, watchdog, leak detection tests) |
| validation / UI workbench | idle → pending → success or error (react-query mutation) | Submitting while pending (button disabled) | `ScenarioValidationWorkbench.test.tsx` |

Neither machine has a `*.flow.json` contract yet; both are enforced by
direct tests rather than a generated model.

## Maturity Ladder

Temporal workflows mature in layers. Do not skip the executable layers
to add a standalone formal document.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, components, callbacks, or jobs. |
| 1 | Inventory | The flow is listed here with owner, source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool is generated from the contract, checked, and replayed by production tests. |

## Production Shape

The required Go or UI files per flow sit at the top of the feature folder,
plus one `generated/` sibling. Everything in `generated/` is codegen output.

Every flow lives in a `flow/` subdirectory next to its consumer with
conventional file names. API domains that own durable lifecycle state use:

```text
api/internal/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.go               # hand: wrapper (package flow)
    flow_test.go                # hand: thin replay delegation (package flow)
    generated/
      model.qnt
      artifact.json
      runtime.go                # package generated
      replay.go
```

UI features that own client-side modes use:

```text
ui/src/features/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.ts               # hand: wrapper
    fixtures.ts                 # hand: replay fixtures
    flow.test.ts                # hand: thin replay delegation
    generated/
      model.qnt
      artifact.json
      runtime.ts
      replay.helper.ts
```

Every flow uses the same file names. The `flow/` directory IS the unit;
the contract no longer declares any output paths or module names.

The workflow owns state/status values, events, `Transition`, and
`CheckInvariants`. It should be pure or nearly pure. Effects live
outside the workflow behind seams: repositories, the executor, the
Code Facts client, clocks, timers, HTTP clients, or UI API modules.

The `*.flow.json` contract is the source of truth. Level 5 generated
Quint models, formal artifacts, and Go/TypeScript declarations are
checked-in source artifacts for reviewability, but they are refreshed
and checked by the `flow-verifier` scenario CLI; the
scenario lifecycle runs `make temporal-models` (which calls
`flow-verifier verify check`) before the normal test
suite. A Quint file by itself is not accepted: the model must typecheck,
test, verify named invariants, emit deterministic artifacts, and those
artifacts must replay against the production Go/TypeScript transition
functions.

The generated declarations keep state/event topology and formal
freshness metadata out of hand-maintained test lists. They also provide
pure status-transition helpers generated from the `*.flow.json`
transition matrix. For TypeScript flows, the same declarations can own
the discriminated state/event union shape and replay fixture contract.
Production workflow wrappers call those helpers for abstract validity
and next-status outcomes, while keeping payload validation, side-effect
orchestration, and rich state construction in hand-authored code. API
replay tests get expected paths, hashes, invariants, and generated checks
from `generated/<folder>/runtime.go`; UI replay tests import the same metadata
from `generated/<folder>/runtime.ts`. The generated `replay.{go,helper.ts}`
files own the assertion calls; the hand-authored top-level test simply binds
the wrapper's transition function and the fixtures and invokes
`RunReplay`/`runFormalReplay` once.

Formal artifacts use schema v6 coverage metadata. Matrix completeness,
terminal transition checks, named trace coverage, and generated MBT trace
coverage are separate fields. Do not treat generated trace
`allPairsCovered` as required proof of correctness; replay tests require
the complete transition matrix and named traces, while generated trace
coverage reports how much the model explorer happened to visit.

Schema v6 `flow.json` files carry no path or module information. The
`replay` block declares only `transition.function` (plus
`transition.statusAccessor` for TS or `transition.stateType` /
`transition.statusField` for Go). Everything else is derived from the
flow directory.

Go flows emit `flow/generated/replay.go` and require a hand-authored
`flow/flow_test.go` (package `flow`) that calls `generated.RunReplay`.
TypeScript flows emit `flow/generated/replay.helper.ts` and require a
hand-authored `flow/flow.test.ts` that calls
`runFormalReplay({ transition, fixtures })` at module top level.
`flow-verifier verify check` byte-compares every generated file and runs an
AST-level lint over the hand-authored test, so a silent bypass — missing
import, stubbed transition, or call buried inside a guarded block —
fails the check.

To scaffold a new flow:

```bash
flow-verifier flows new "ui/src/features/<feature>" --flow-id "<flow-id>" --lang ts --root .
flow-verifier flows new "api/internal/<domain>"     --flow-id "<flow-id>" --lang go --root .
```

The scaffold writes the hand-authored files and immediately runs
`generate`, so `check` is green from the moment it returns.

To add or rename a state/event:

1. Edit the owning `*.flow.json`.
2. Regenerate that flow with `flow-verifier verify run --flow <flow-id>`.
3. Update only payload-specific wrapper branches that need new runtime
   data; the abstract transition table is generated.
4. Update the UI replay fixture module. The generated formal replay fixture
   interface should make missing state/event fixtures a type error.
5. Run `make temporal-models` and the scenario tests.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Executed command lifecycle | Terminal-state and cleanup rules live only in `api/internal/executor`; a regression in teardown ordering is caught by tests, not by a checked model. | Promote to Level 2+ with a `flow/` directory under `api/internal/executor/` if the lifecycle grows (retries, partial re-runs). |
| Workbench request state | Client state is a react-query mutation; adding cancel, stale-run detection, or multi-run comparison would introduce real modes. | Add `ui/src/features/validation/flow/` when a second mode appears. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md#temporal-workflow-tests) — matrix and trace testing
