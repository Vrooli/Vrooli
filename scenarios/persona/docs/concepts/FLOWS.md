# Flows — Persona

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

`health` is a stateless reporting domain and ships no workflows.
Persona has four genuinely stateful flows. Everything else in the
scenario is plain CRUD with no ordering constraints and deliberately
does not appear here.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Handoff lifecycle | handoffs | A flow meets a wall a machine must not cross | A human completes the step and the flow resumes from its checkpoint | Ordered states, expiry, cancellation, stale completion, resumption after the originating run has died | **Level 3** target (matrix + traces) |
| Act-as authorization | access | An agent asks to act as a persona | An act-as session, or a journaled refusal | Short ordered sequence with a hard fail-closed branch on an unreachable authority | **Level 3** target |
| Code retrieval | channels | A flow needs a one-time code | The code returned exactly once, or a named unavailability | Retries against a moving target, expiry mid-read, adapter fallback rules | **Level 2** target (state/event matrix) |
| Document release | documents | A handoff needs an identity document | The document reaches the named handoff, with a permanent release record | Two-party ordering across a scenario boundary; must be idempotent per handoff | **Level 3** target |

## Flow Details

### Handoff lifecycle — `handoffs`

The scenario's defining flow. A wall is not an error; it is a state.

- **Trigger**: any domain reports that the next step requires a human —
  a CAPTCHA, a biometric prompt, a government-ID upload, a review queue,
  or an operator decision.
- **Inputs**: the originating persona, the wall kind, everything already
  completed (the checkpoint), and what specifically the human must do.
- **Steps**: `open` captures the checkpoint and the required action →
  `deliver` attempts a relay if one is configured, and always enqueues
  locally → the human acts → `complete` records the outcome →
  `resume` hands control back at the checkpoint.
- **Outputs**: a completed handoff, a resumable checkpoint, and journal
  rows for every transition.
- **Failure modes**: the relay is absent or down (the queue still
  serves); the human never acts (the handoff expires, and expiry is a
  first-class terminal state, not a silent drop); the originating run
  has died before completion (resumption must not require it).
- **Retry/cancel**: delivery retries with backoff; the handoff itself is
  never auto-retried, because re-presenting a human action they may have
  already taken elsewhere is worse than waiting. Cancellation is
  explicit and terminal.
- **Key rule**: a handoff never instructs a human to defeat a
  verification control. It routes them to satisfy it legitimately.
- **Tests**: full state/event matrix, replay traces for the
  relay-absent, expiry, and dead-originator paths.

### Act-as authorization — `access`

- **Trigger**: an agent requests to act as a named persona.
- **Inputs**: the run's identity token, the requested persona, the
  requested entitlement.
- **Steps**: verify the token with `agent-manager` → confirm the ACL
  admits this human → intersect requested entitlement with what the
  persona grants → open an act-as session → journal.
- **Outputs**: a session and a resolution payload, or a journaled
  refusal carrying its reason.
- **Failure modes**: authority unreachable, token invalid or expired,
  ACL denies, entitlement exceeds the grant.
- **Key rule**: **every failure path refuses.** There is no degraded
  evidence grade and no override flag. An unverifiable caller cannot act
  as anyone.
- **Tests**: matrix over (token valid/invalid/unreachable) × (ACL
  allow/propose-only/deny), plus an attenuation test proving a child run
  can never hold more than its parent.

### Code retrieval — `channels`

- **Trigger**: a flow reports it is waiting on a one-time code.
- **Inputs**: the persona, the channel binding, a time window, and an
  expected sender or shape where one is known.
- **Steps**: resolve the adapter → fetch the credential from
  `secrets-manager` → poll the channel within the window → match the
  code → return it once → record the fetch without the value.
- **Outputs**: the code, returned exactly once, with its expiry.
- **Failure modes**: no code arrives in the window; several codes match
  and the newest is ambiguous; the code expires between read and use;
  the adapter is unavailable.
- **Retry/cancel**: bounded polling inside the caller's window;
  cancellation is immediate.
- **Key rule**: **never fall back to another persona's route.** An
  unavailable adapter is a named failure, because a code fetched as the
  wrong identity is worse than no code at all.
- **Tests**: state/event matrix covering arrival, no-arrival, ambiguity,
  and expiry-mid-read.

### Document release — `documents`

- **Trigger**: an open handoff declares it needs a bound document.
- **Inputs**: the persona, the document binding, the target handoff id.
- **Steps**: confirm the handoff exists and is open → confirm the
  binding belongs to this persona → ask `document-manager` to release
  into that handoff → write a permanent release record → journal.
- **Outputs**: the document available to the named handoff only, plus a
  release record that outlives it.
- **Failure modes**: handoff closed or unknown; binding belongs to a
  different persona; `document-manager` unreachable.
- **Retry/cancel**: idempotent per (binding, handoff) — a repeated
  release is a successful no-op rather than a second disclosure.
- **Key rule**: **there is no agent-readable read path.** Release
  targets a handoff; no scope returns bytes to an agent.
- **Tests**: cross-boundary ordering, idempotency, and a negative test
  asserting no request shape returns document content.

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

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| handoffs / lifecycle | `open` → `delivered` → `completed`; `open`/`delivered` → `expired`; `open`/`delivered` → `cancelled`. Terminal: `completed`, `expired`, `cancelled`. | Any terminal → any state. `open` → `completed` without a recorded human actor. `expired` → `delivered`. Resumption against a non-`completed` handoff. | `flow.json` contract, generated Quint model, replay tests. |
| access / act-as | `requested` → `verified` → `granted` → `closed`; `requested` → `refused`; `verified` → `refused`. Terminal: `closed`, `refused`. | `requested` → `granted` (skipping verification). `refused` → any state. Any transition out of a terminal state. | `flow.json` contract, generated Quint model, replay tests. |
| documents / release | `requested` → `released` → `recorded`; `requested` → `refused`. Terminal: `recorded`, `refused`. | `released` without an open target handoff. `recorded` → `released` (double disclosure). `refused` → `released`. | `flow.json` contract, generated Quint model, replay tests. |
| channels / retrieval | `waiting` → `matched` → `delivered`; `waiting` → `timed_out`; `matched` → `expired`. Terminal: `delivered`, `timed_out`, `expired`. | `delivered` → `delivered` (a code is single-use). `timed_out` → `matched`. Adapter substitution mid-flow. | `flow.json` contract, generated Quint model, replay tests. |

The recurring invariant across all four: **terminal states are
terminal**, and every refusal is terminal. Nothing in this scenario
retries its way from a refusal into a success.

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
| Persona retirement | Medium — retiring a persona can orphan accounts it created, and the failure is silent and permanent. | Model once `accounts` (P1) holds recovery paths; retirement is deliberately blocked until then (`PSN-P2-002`). |
| Enrolment preparation | Medium — a multi-wall enrolment produces several handoffs whose ordering matters to the target. | Model at `PSN-P1-008`, once one real enrolment has been observed end to end. |
| Attestation exchange | Low — verification of a counterparty's attestation is stateless today. | Revisit if `PSN-P2-004` introduces challenge/response. |
| Paid human handoff | Medium — introduces a second party and a payment, so a failed completion has a money consequence. | Model with `treasury` at `PSN-P2-003`; not before. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
