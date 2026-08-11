# Flows — Scenario to iOS

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

`health` is a stateless reporting domain and ships no workflows. List
each real stateful flow your domains add below, with its owner, trigger,
outcome, statefulness, and validation level.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Project generation | projects | Operator or agent requests an iOS app for a scenario at a revision. | A buildable Xcode project with a derived bundle identifier. | Ordered, idempotent per (scenario, revision, template). Regeneration supersedes; it does not mutate in place. | Level 3 — matrix over generate/regenerate/supersede. |
| Build-host resolution | targets | Any build or simulator cell is selected. | A node that fills the build-host role, or `unavailable` with the missing requirement. | Ordered probe: node online → Xcode present → version at floor → runtimes installed → GUI session → signing identity. Each failure has its own distinct reason. | Level 4 — the `unsupported` versus `unavailable` distinction is the invariant. |
| Remote artifact build | builds | A generated project plus a requested output kind, plus a resolved build host. | A signed artifact reference plus a recorded SDK-floor assertion and executing-node identity. | Ordered with a fail-closed assertion gate before packaging. Cancellable; cancellation must reach the remote node, not just the local record. | Level 5 — remote cancellation and orphaned-remote-work are the failure modes worth model-checking. |
| Conformance journey run | journeys | A validation cell dispatches to the `Driver`. | A chaptered journey with assertions, capture references, strategy attribution, and a disposition. | Long-running, lease-scoped, cancellable. Every chapter has bounded readiness and settle policies; an exceeded bound is a failure, never a longer wait. | Level 5 — lease-holding, bounded waits, and strategy attribution all need replay proof. |
| Matrix run and gate | releases | Operator or CI starts a matrix over selected cells. | A fail-closed gate verdict plus reference-only `TargetVerdict` emission. | Immutable once started; wait, abort, rerun, and compare come from the spine. Promotability is evaluated per contributing cell. | Level 4 — immutability plus the mirror-evidence non-promotion rule. |
| Distribution attempt | distribution | An artifact plus a selected channel. | An upload receipt, or `unavailable` naming the blocking rung. | Per-channel; one channel's failure never changes another's availability. Retries are idempotent against the receipt. | Level 3 — per-channel matrix. |
| Readiness probe | readiness | Scheduled, or on demand from CLI/UI. | Current rung states with next actions. | Stateless read modelled as a flow because rung order matters — certificates cannot report ready while enrollment is not. | Level 2 — ordering invariant. |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| targets / Build-host resolution | `unresolved → probing → resolved`; terminal `unavailable`, `unsupported`, `validate_only`. | `unsupported → unavailable` or the reverse — they are different kinds of answer and neither converts into the other. `resolved` when the toolchain is below the SDK floor; that is `validate_only`. | `targets.flow.json` contract, generated model, replay traces per failure reason. |
| builds / Remote artifact build | `queued → dispatching → assembling → asserting → signing → complete`; terminal `failed`, `cancelled`. | `asserting → signing` when the SDK-floor assertion did not hold. `cancelled → complete`. Any path to `complete` that skips `asserting`. Local `cancelled` while remote work continues. | `builds.flow.json` contract, generated Quint model, replay traces including remote cancellation and node loss mid-build. |
| journeys / Conformance run | `pending → lease_held → running → settling → complete`; terminal `failed`, `cancelled`, `unavailable`. | `pending → running` without a held `device-control` lease. `complete` while a chapter is still unsettled. Promotion of `unavailable` to `complete`. | `journeys.flow.json` contract, generated Quint model, replay traces including lease loss mid-run. |
| releases / Matrix run and gate | `selected → dispatching → collecting → gated`; terminal `passed`, `failed`, `aborted`. | Mutation of a selection after `dispatching`. `gated → passed` while any required cell is `unavailable` or `unsupported`. `gated → passed` where the only supporting evidence came from a non-promotable strategy. | Spine-owned immutability plus a local gate-composition and promotability contract. |
| distribution / Distribution attempt | `blocked → ready → uploading → published`; terminal `failed`. | `blocked → uploading` without the gating rung satisfied. Duplicate publish for one receipt. | `distribution.flow.json` contract; idempotency keyed on the receipt. |

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

Three (Go) or four (UI) files per flow at the top of the feature folder,
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
outside the workflow behind seams: repositories, BlobStore, clocks,
timers, HTTP clients, or UI API modules.

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
flow-verifier flows new ui/src/features/<feature> --flow-id <flow-id> --lang ts --root .
flow-verifier flows new api/internal/<domain>     --flow-id <flow-id> --lang go --root .
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
| None yet. | Generated scaffold. | Add real scenario workflows when domains have stateful behavior. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
