# Flows — Music Tools

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
| Model install | models | Operator or CLI requests a model | Weights verified and available | Staged and resumable; refuses below the disk floor; checksum failure leaves the model unavailable | Level 3 |
| GPU-bearing operation | capacity + jobs | Any operation needing the card | Artifact plus the applied profile rung | Claim, admit or queue or degrade, execute, release; release on failure and on crash | Level 5 |
| Composition job | composition + jobs | Caption and optional lyrics submitted | Audio with provenance | Long-running, progress-reporting, cancellable, survives client disconnect | Level 5 |
| Track decomposition | analysis | Consumer requests attributes | Structured description | Partial success when a runtime is unavailable; never silently incomplete | Level 4 |
| Derived-artifact eviction | storage | Budget exceeded | Space reclaimed | LRU over regenerable artifacts only | Level 3 |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

### GPU-bearing operation — `capacity` + `jobs`

The flow every heavyweight operation passes through. It exists because the card is
shared and no two heavyweight models co-reside.

- Owner domains: `capacity` (admission) and `jobs` (lifecycle).
- Trigger: any composition, transformation, or structure operation.
- Steps:
  1. Resolve the model, honouring the configured licence lane.
  2. Declare ordered profile rungs — variant, planner size, precision, batch.
  3. Claim capacity through the control-plane broker.
  4. On denial, queue with a stated reason. On contention, step down a rung.
  5. Execute under an exclusive lease, heartbeating.
  6. Release, recording the applied rung with the result.
- Illegal transitions: executing without a granted claim; completing without
  releasing; degrading without recording the rung.
- Failure modes: admission denied (queue); lease lost (fail explicitly, release);
  process crash (broker reconciliation expires the claim).
- Why Level 5: the release path must hold under crash and cancellation, and a
  leaked lease starves every other tenant on the host.

### Composition job — `composition` + `jobs`

- Trigger: caption and optional lyrics submitted via API or CLI.
- Steps: validate and compile the caption, acquire the GPU lease above, generate,
  write through the BlobStore seam with provenance, release.
- Statefulness: server-owned; survives client disconnect; progress observable;
  cancellation releases the lease rather than orphaning it.
- Failure modes: composition runtime stopped (fail explicitly — never silently
  substitute a cloud provider); budget exhausted (evict, then proceed).

### Track decomposition — `analysis`

- Trigger: a consumer asks for the attributes of a track.
- Steps: deterministic measures first (cheap, exact, no GPU), then embeddings from
  the resident pool, then structure and beats under an exclusive lease.
- Statefulness: each layer succeeds or fails independently.
- Key rule: when a runtime is unavailable the result is **explicitly partial**, not
  quietly short. A consumer must be able to tell a missing attribute from an absent
  one.

### Model install — `models`

- Trigger: operator or CLI request.
- Steps: check the declared free-disk floor, download from the declared source,
  verify the checksum, mark installed.
- Illegal transitions: marking installed without verification; beginning a download
  below the disk floor.
- Failure modes: disk floor breach (refuse to start); checksum mismatch (model stays
  unavailable rather than running unverified).

### Derived-artifact eviction — `storage`

- Trigger: a write that would exceed the declared budget.
- Steps: evict least-recently-used regenerable artifacts until the write fits.
- Invariant: **only regenerable artifacts are evictable.** Nothing here is ever the
  only copy of anything.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships an `Attachment upload` flow on the `notes` domain as a
worked Level 5 temporal-workflow vertical slice. Copy its shape for your
own stateful flows, then remove it.

Add this row to the Flow Inventory above:

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Attachment upload | notes | User/CLI uploads a file for a note. | Blob is stored and metadata is persisted. | Stateful upload request with validation and failure paths. | Level 5 workflow tests: matrix, traces, declarative spec, checked Quint model, generated artifacts, and production replay. |

#### Attachment upload

- Owner domain: notes.
- Trigger: multipart upload request from UI or CLI.
- Inputs: note id, file key/name, file bytes, content type, file size.
- Steps:
  1. Parse multipart request.
  2. Validate note id and file metadata.
  3. Store opaque bytes through BlobStore.
  4. Persist attachment metadata through notes repository seam.
  5. Return proto-typed metadata response.
- Outputs: uploaded attachment metadata or typed error response.
- Failure modes: missing note id, missing file, invalid metadata, blob
  write failure, metadata persistence failure.
- Retry/cancel behavior: caller may retry after transport/storage
  failure; duplicate handling belongs to the owning real domain when
  product requirements demand it.
- Tests: `api/handlers/notes/attachments_handler_test.go`,
  `api/internal/notes/attachments_service_test.go`,
  `api/internal/notes/flow/flow_test.go`,
  `ui/src/features/notes/AttachmentUpload.test.tsx`, and
  `ui/src/features/notes/flow/flow.test.ts`.
- Generated subpackages: `api/internal/notes/flow/generated/`
  (`model.qnt`, `artifact.json`, `runtime.go`, `replay.go`) and
  `ui/src/features/notes/flow/generated/` (`model.qnt`, `artifact.json`,
  `runtime.ts`, `replay.helper.ts`).
- Requirements: template starter only.

These example state machines belong in the State Machines table below:

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| notes / attachment upload API | received, bytes_stored, metadata_recorded, failed | metadata before bytes, terminal-state escape, duplicate terminal events | `*.flow.json` contract, generated Quint model, generated formal artifact replay, side-effect cleanup tests |
| notes / attachment upload UI | idle, selected, uploading, succeeded, failed | start before select, stale completion after reset/reselect, retry without file context | `*.flow.json` contract, generated Quint model, generated formal artifact replay, attempt-id stale completion tests |
<!-- EXAMPLE-DOMAIN:notes END -->

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| capacity / GPU lease | `requested` → `queued` → `granted` → `degraded`* → `released`; terminal `denied`, `expired` | Executing without `granted`; completing without `released`; `degraded` without recording the applied rung | `*.flow.json` contract, generated Quint model, replay tests; broker reconciliation expires leaked leases |
| jobs / long operation | `submitted` → `running` → `succeeded` \| `failed` \| `cancelled` | Any terminal state that leaves a lease held; `succeeded` without a persisted artifact | Contract plus replay tests over client-disconnect and crash traces |
| models / install | `declared` → `downloading` → `verifying` → `installed`; terminal `refused`, `failed` | `installed` without passing `verifying`; `downloading` below the declared disk floor | Contract tests; checksum verification is a hard gate |
| storage / eviction | `within-budget` → `over-budget` → `evicting` → `within-budget` | Evicting a non-regenerable artifact | Contract test asserting only regenerable kinds are evictable |

\* `degraded` is a re-entrant rung step, not a terminal state.

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

These are known but deliberately unmodelled today:

- **Style compilation** — currently a pure function from style to caption plus
  parameters, with no ordered states. It becomes a flow only if styles gain
  validation or derivation steps.
- **Adapter training** — a P2 target gated on hardware the reference host does not
  have. It will be a long-running stateful flow when it exists; modelling it now
  would be speculative.
- **Multi-track composition** — depends on an upstream capability whose availability
  on the fitting model variant is unresolved. See
  [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).
- **Batch decomposition** — orchestration across many tracks belongs to the
  consumer, which owns progress and resumability. This scenario stays per-track.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
