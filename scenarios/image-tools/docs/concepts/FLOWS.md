# Flows — Image Tools

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

All flows below are **planned (pre-implementation)**. No formal Quint
model exists yet; each is at maturity Level 1 (inventoried) and targets
the Level it lists when its domain is built.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Single-op async job lifecycle | jobs | User/CLI/UI submits an operation. | Output stored, status terminal (succeeded/failed/canceled). | Highly stateful: queued→running→terminal, GPU-serialized, cancellable. | Target Level 5: matrix, traces, declarative spec, checked Quint model, replay. |
| Model install/enable/select lifecycle | models | User installs/enables a model or selector runs at op time. | Model available + selected for the probed host. | Stateful install + enable + selection states. | Target Level 4–5. |
| Backend fallback chain | backends | Op cannot run on the current tier. | Op runs on next available tier with user-visible messaging. | Stateful tier transitions (Local-GPU→Local-CPU→BYOK). | Target Level 4–5. |
| Recipe replay | recipes | User runs a saved recipe. | Each op-stack step executes in order; outputs produced. | Sequential multi-step, per-step failure handling. | Target Level 3–4. |
| Watch-folder trigger | automation | New file appears in a watched folder. | Configured recipe/op submitted as a job. | Debounced trigger → job submission. | Target Level 3–4. |
| Attachment upload (**template example, remove**) | notes | User/CLI uploads a file for a note. | Blob stored, metadata persisted. | Stateful upload request with failure paths. | Template Level 5 reference. |

## Flow Details

### Single-op async job lifecycle

- Owner domain: jobs (operation domains submit; `backends` resolves the
  provider; `storage` persists output).
- Trigger: an operation submission from CLI, UI, or API.
- Inputs: op id + parameters, source blob handle(s), optional save-location
  override, optional signed callback URL.
- Steps:
  1. Validate op + parameters; run ingestion safety guard on inputs.
  2. Create a server-owned job record; return **job-id + ETA up front**.
  3. Enqueue: heavy GPU ops go to the serializing queue; cheap CPU ops run
     concurrently.
  4. Resolve provider via `backends` (apply fallback ladder if needed).
  5. Execute; stream progress over SSE (UI) — clients never poll.
  6. Persist output via `storage`; transition to a terminal state.
  7. If a callback URL was supplied, deliver best-effort signed POST.
- Outputs: stored output blob + terminal job status, or typed error.
- Failure modes: invalid params, oversized/decompression-bomb input, no
  available provider/model, GPU OOM, cancellation, storage write failure.
- Retry/cancel: caller may cancel a running job; the CLI **blocks once**
  on a wait verb (no polling); callbacks retry on failure.
- Requirements: OT-P0-009 (queue), OT-P0-011 (fallback), OT-P0-010
  (storage), OT-P1-006 (callbacks).

### Model install/enable/select lifecycle

- Owner domain: models.
- Trigger: user install/enable/disable/remove, or the selector running at
  op time.
- Steps:
  1. List/search registry entries with size, hardware-fit, license, labels.
  2. Disk-space check; checksummed opt-in download (or register local path).
  3. Mark installed → enabled.
  4. At op time the selector reads the host probe (system-monitor /
     hostinventory) and picks the best-fit enabled model, honoring per-op
     default and user override.
- Failure modes: insufficient disk, checksum mismatch, no host-fit model
  (surface "needs ≥X GB VRAM"), removal of a model in use.
- Requirements: OT-P0-006, OT-P0-007, OT-P0-008.

### Backend fallback chain

- Owner domain: backends.
- Trigger: the selected tier cannot run the op (no GPU, insufficient VRAM,
  provider unavailable).
- Steps: attempt Local-GPU → on failure Local-CPU (guaranteed CPU fallback
  model) → on failure BYOK-cloud (after a pre-op cost estimate the user
  accepts). Each transition emits explicit user-visible messaging with the
  hardware-fit reason.
- Failure modes: no standalone provider (prevented at registration), BYOK
  unavailable / over budget, user declines cost estimate.
- Requirements: OT-P0-005, OT-P0-011.

### Recipe replay

- Owner domain: recipes.
- Trigger: user runs a saved recipe (UI op-stack or CLI pipeline — one
  representation).
- Steps: load recipe graph → execute each step in order (each step is a
  normal op submitted through `jobs`) → chain outputs to next inputs →
  produce final output.
- Failure modes: missing input, a step's op/model unavailable, mid-pipeline
  failure (stop with partial state reported).
- Requirements: OT-P1-004.

### Watch-folder trigger

- Owner domain: automation.
- Trigger: a new file lands in a watched folder.
- Steps: detect → debounce (configurable) → submit the configured recipe or
  single op as a job → route output per configuration → optional callback.
- Failure modes: unreadable file, debounce churn, output-routing failure.
- Requirements: OT-P1-005.

### Attachment upload

> Template example — remove with the `notes` domain during implementation.

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

## State Machines

The following are the **planned** state machines for image-tools' real
flows. They are not yet modeled in Quint; they define the target topology.

Single-op async job lifecycle (jobs):

```text
  submit
    │
    ▼
 queued ──► running ──► succeeded   (terminal)
    │          │
    │          ├──► failed          (terminal)
    │          └──► canceled        (terminal)
    └──► canceled                   (terminal, cancel while queued)
```

Backend fallback chain (backends):

```text
 local_gpu ──fail──► local_cpu ──fail──► byok_cloud ──► (run | give_up)
     │                  │                     ▲
     └── run            └── run               └── only after cost-estimate accepted
```

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| jobs / single-op job (API) | queued, running, succeeded, failed, canceled | running before queued, escape from a terminal state, succeeded after canceled, duplicate terminal events | Planned `*.flow.json` contract + generated Quint model + replay (target Level 5). |
| jobs / progress (UI) | idle, submitted, running, succeeded, failed, canceled | progress before submit, stale completion after cancel/resubmit, success after cancel | Planned `*.flow.json` + generated model + attempt-id stale-completion tests. |
| backends / fallback chain | local_gpu, local_cpu, byok_cloud, ran, gave_up | byok_cloud before cost-estimate accepted, skipping CPU when a CPU model exists, re-entering a failed tier | Planned `*.flow.json` + generated model + replay. |
| models / install+select | absent, downloading, installed, enabled, disabled, selected | enabled before installed, selected when disabled, removal while in use | Planned model + repository/selector tests. |
| automation / watch-folder | watching, debouncing, submitted | submit before debounce settles, re-trigger on same unchanged file | Planned model + debounce tests. |
| notes / attachment upload API (template, remove) | received, bytes_stored, metadata_recorded, failed | metadata before bytes, terminal-state escape, duplicate terminal events | `*.flow.json` contract, generated Quint model, replay, side-effect cleanup tests. |
| notes / attachment upload UI (template, remove) | idle, selected, uploading, succeeded, failed | start before select, stale completion after reset/reselect, retry without file context | `*.flow.json` contract, generated Quint model, replay, attempt-id stale-completion tests. |

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

image-tools is pre-implementation, so no flow has reached this production
shape yet. When the domains are built, the single-op job lifecycle (jobs)
is the first flow that should be taken to the full Level-5 production shape
below, since it carries the most lifecycle risk (durability across client
disconnect, GPU serialization, cancellation, stale completion). The
backend fallback chain and watch-folder trigger follow.

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

All image-tools flows are currently unmodeled (Level 1, inventoried only)
because the scenario is pre-implementation. They are listed in the Flow
Inventory above with owners and target levels.

| Flow | Risk | Next Step |
|---|---|---|
| Single-op async job lifecycle | High — durable server-owned runs, GPU serialization, cancel/stale completion. | Model to Level 5 first when `jobs` is built. |
| Backend fallback chain | Medium-high — tier transitions + BYOK cost gate must be correct and well-messaged. | Model to Level 4–5 when `backends` is built. |
| Watch-folder trigger | Medium — debounce + trigger-once correctness. | Model to Level 3–4 when `automation` is built. |
| Recipe replay | Medium — sequential multi-step with partial-failure semantics. | Model to Level 3–4 when `recipes` is built. |
| Model install/select | Medium — install/enable/select ordering, removal-while-in-use. | Model to Level 4 when `models` is built. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
