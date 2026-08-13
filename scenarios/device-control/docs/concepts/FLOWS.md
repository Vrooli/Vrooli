# Flows — Device Control

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
| Capability probe | devices | Scheduled sweep, bridge fleet change, or explicit refresh. | A capability snapshot per device recording each capability as available or unavailable with its missing prerequisite. | Stateful per device; a probe that cannot complete records unavailable rather than retaining the prior snapshot. | Level 3 — matrix over capability × probe outcome. |
| Lease acquisition | sessions | A consumer requests control of a device. | An exclusive, expiring lease, or a refusal naming the current holder. | Ordered: requested → held → (renewed) → released \| expired \| killed. Terminal states are absorbing. | Level 4 — declarative contract; concurrency is the risk. |
| Flow validation | flows | A flow is saved, or a run is requested. | A capability gap report: the set of required capabilities this strategy cannot satisfy. | Stateless pure function over (flow, capability declaration). | Level 3 — matrix over step kind × declared capability. |
| Flow run | flows | Operator, agent, delivery ramp, or schedule. | A chaptered evidence record with a terminal disposition. | Ordered: validated → leased → running → terminal (passed \| failed \| aborted \| unavailable). Bounded waits inside each step. | Level 5 — this is the flow that produces release-relevant evidence. |
| Target resolution | flows | Each step that names a target by intent. | A concrete coordinate or element handle, plus the rung used and its confidence. | Stateless per attempt; falls down the ladder, never up. | Level 3 — matrix over rung × strategy tier. |
| Agent run | agent | `device-control agent run --goal`. | A terminal goal outcome plus a recorded step sequence eligible for promotion. | Bounded: step count, cost ceiling, lease scope. Abortable at any point. | Level 4 — bounds are the safety property. |
| Agent-run promotion | agent | Operator promotes a successful agent run. | A deterministic flow whose replay contains no `ai.*` step. | Stateless transform over a completed run's step record. | Level 3 — replay equivalence. |

### Target resolution

The target resolver is exposed through `POST /api/v1/flows/resolve-target`
for the flow executor and CLI surfaces. The caller submits a frame and target
intent; device-control decodes and downsizes the frame before sending it to
ai-gateway with role `locate.visual`. The gateway response is canonical
normalized bounds. Device-control converts those bounds against the original
capture dimensions, preserving device coordinates even when the submission
was downscaled.

The evidence sequence is ordered and redaction-safe:

1. `attempt_vision` records the vision rung and submitted dimensions.
2. `resolved` records the selected rung and confidence, or `fallback` records
   the reason for selecting the lower `visual-anchor` rung.
3. `unresolved` records a typed reason when neither the gateway route nor a
   caller-owned anchor can resolve the target.

No event contains frame bytes, screen text, provider URLs, model slugs, or
credentials.

The current resolution order is deterministic semantic tree, persisted visual
anchor, then vision. A semantic hit is asserted without an ai-gateway call;
successful vision resolutions may be promoted to an anchor with normalized
bounds and a frame checksum. Flow steps use normalized pointer coordinates by
default. The typed vocabulary includes swipe, long-press, double-tap, drag,
fling, scroll-to, and capability-gated pinch; pixel coordinates are an
explicit non-portable escape hatch.

Session-scoped recordings carry a claim class (`static`, `transition`, or
`animation`) and a measured effective frame rate. A recording below its class
minimum is `degraded`, never silently treated as passed. Device state changes
are owned by the lease and restored in reverse order when a session ends.

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| sessions / lease | requested, held, renewed, released, expired, killed | Grant while another lease is held; renew after expiry; any transition out of a terminal state; verb dispatch without `held`. | `*.flow.json` contract, generated Quint model, concurrency tests asserting the second claim is refused rather than queued. |
| flows / run | validated, leased, running, passed, failed, aborted, unavailable | Run before validation; run without a held lease; terminal-state escape; a step completing after abort; an exceeded bounded wait resolving as success. | `*.flow.json` contract, generated Quint model, replay tests over recorded runs. |
| flows / resolution | attempt_semantic, attempt_anchor, attempt_vision, resolved, unresolved | Climbing back up the ladder after a lower rung was used; resolving via a rung the strategy did not declare; recording `resolved` without a rung and confidence. | Matrix tests over rung × strategy tier; evidence assertion that the rung is always recorded. |
| agent / run | planning, acting, observing, terminal | Acting after a bound is exhausted; acting without a held lease; continuing after abort. | Bound-exhaustion tests, abort tests, audit completeness assertion. |

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
