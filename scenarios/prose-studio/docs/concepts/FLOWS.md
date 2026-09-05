# Flows — Prose Studio

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
| Candidate round | generation | A session requests k candidates against a resolved profile. | A round holding k candidates, each measured at birth, each eligible or ineligible with a named reason. | Bounded, single-shot per call; a partial provider response fails the round rather than presenting a short set as complete. | Level 3 — matrix + traces. |
| Session convergence | sessions | An operator or agent opens a session against a profile and query. | One committed candidate plus an append-only history of what was rejected and why. | Stateful, long-lived, resumable; pin/reject state must survive reroll; abandonment is terminal but retained. | **Level 5 — the illegal transitions are the product.** |
| Declaration registration | declarations | Startup scan, or an explicit reindex verb. | Every declaration file in the fleet registered, invalid, or unregistered. | Idempotent; must never block startup; collisions must not resolve. | Level 4 — declarative contract. |
| Document composition | documents | An operator creates a document against a profile and brief. | An assembled document of committed sections with coherence measured across them. | Stateful and ordered; feasibility gates entry; a reroll of a committed section invalidates downstream context snapshots. | **Level 5 — ordering and staleness are the risk.** |
| Profile feasibility check | documents | Profile validation, and again pre-call per section. | Accept, or a named refusal identifying the worst-case section and the resolved model's window. | Stateless, deterministic; static check must target the worst section, not the first. | Level 4 — declarative contract. |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| sessions / convergence | `open`, `generating`, `awaiting_choice`, `committed`, `abandoned` | Commit before any round exists; commit an ineligible candidate; commit a candidate from a different session; reroll after commit; mutate a pinned candidate; abandon then commit; two commits on one session | `*.flow.json` contract, generated Quint model, generated formal artifact replay, stale-completion tests keyed by round attempt id |
| generation / round | `requested`, `dispatched`, `returned`, `measured`, `gated`, `failed` | Gate before measure; measure before return; present a partial set as complete; write a candidate with no provenance; write a measurement computed after choice time | `*.flow.json` contract, generated Quint model, replay tests |
| documents / composition | `draft`, `outline_pending`, `outline_committed`, `sections_in_progress`, `assembled`, `abandoned` | Start a section before the outline commits; assemble with an uncommitted section; commit a section whose feasibility check never ran; leave a downstream `context_snapshot` unmarked after an upstream section is rerolled | `*.flow.json` contract, generated Quint model, replay tests over multi-section traces |
| declarations / registration | `discovered`, `parsed`, `registered`, `invalid`, `unregistered` | Resolve a key collision by last-writer-wins; hard-delete a record whose file disappeared; block startup on a parse error; accept an API write to a record marked `authority: file` | `*.flow.json` contract, generated Quint model, replay tests |

**Why two flows target Level 5.** Session convergence and document composition
are the two places where an illegal transition silently produces plausible
output rather than an error. A commit of an ineligible candidate, or an assembly
that includes a section whose upstream context changed underneath it, both yield
a document that looks finished and is wrong. That is the exact failure class
formal modelling exists for, and it is why the flow contract — not a handler
comment — is the enforcement.

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
| Style version freeze-on-reference | A style version mutated after a committed output referenced it silently changes what that output claims to have been generated from. Low frequency, high blast radius. | Model as a Level 4 contract once `styles` has a second writer; the P0 enforcement is a write-path check plus tests. |
| Consumer declaration promotion | One-way export from an operator-authored record to a declaration file has a window where both a row and a file claim the same key. | Level 3 traces at P0; promote if a second promotion path appears. |
| Learned-configuration selection (P2) | Not built. A configuration bandit that drifts toward comfortable settings would degrade diversity invisibly, with no error anywhere. | Do not model before it exists; when it does, its acceptance gate is a measured diversity floor, not a transition contract. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
