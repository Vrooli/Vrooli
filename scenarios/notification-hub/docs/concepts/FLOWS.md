# Flows — Notification Hub

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
| Notification lifecycle | notifications | A caller submits a send request, or a scheduled notification comes due. | The notification reaches a terminal state and the caller can see which. | Eight states, four of them terminal; holds, cancellation, and duplicate suppression. | Level 4 — declarative `*.flow.json` contract with matrix and trace tests. |
| Delivery attempt | delivery | Routing produces a decision for one channel. | A receipt, or a stated terminal failure. | Retry with backoff under a budget; stale completion after cancellation. | Level 4 — contract plus injected-clock retry tests. |
| Quiet-hour hold and release | notifications, routing | A notification arrives inside a quiet window. | The notification is held and later released, or delivered immediately because it is critical. | Timed hold with a release trigger; survives restart. | Level 3 — matrix and trace tests across timezone and midnight-crossing cases. |
| Duplicate suppression | notifications | A send request carries a dedupe key already seen inside its window. | The duplicate is collapsed onto the original and marked suppressed. | Windowed, per recipient. | Level 3 — matrix tests. |
| Relayed delivery (P1) | relay, delivery | Routing decides this host cannot serve the channel. | A delivery performed by a fleet node, with the same receipt shape as a local one. | Dispatch acknowledgement, node absence, timeout, duplicate dispatch. | Level 4 — contract with a stubbed dispatch seam. |
| Event ingress (P1) | ingress | An inbound webhook arrives from `vrooli-events` or a third party. | A notification request, or an explicit no-match. | Idempotent on redelivery via receipts. | Level 3 — matrix tests against a synthetic caller. |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

### Notification lifecycle — `notifications`

The spine of the scenario. Everything else is a step inside it.

- **Trigger** — an authenticated send request from a scenario, an agent,
  the CLI, or the UI; or the scheduler finding a notification whose
  `scheduled_for` has passed.
- **Inputs** — recipient, title, body, urgency, sensitivity, optional
  dedupe key and window, optional scheduled time, optional link.
- **Ordered steps**
  1. Validate and persist as `accepted`. The caller gets an id here, and
     the id is durable before any delivery is attempted. A caller that
     received an id can always find out what happened.
  2. Check the dedupe key against the recipient's window. A hit
     terminates this notification as `suppressed` and records which
     notification absorbed it.
  3. Ask `routing` for a decision. Routing reads recipient preferences,
     quiet windows, channel availability, and the sensitivity label.
  4. If the decision is to hold, move to `held` with a release time.
     Otherwise move to `routed`.
  5. Create one delivery row per selected channel and move to
     `delivering`.
  6. When deliveries settle, move to `delivered` if any channel
     succeeded, or `failed` if all of them exhausted their budget.
- **Outputs** — a terminal state, a transition history, and one delivery
  row per attempted channel.
- **Failure modes** — storage unavailable at step 1 rejects the request
  rather than accepting something that cannot be persisted; no reachable
  channel at step 3 fails the notification immediately with a stated
  reason rather than leaving it pending forever.
- **Retry and cancel** — retries belong to the delivery flow, not here.
  Cancellation is permitted from `accepted`, `held`, and `routed`, and
  is refused once a delivery has left for a provider, because the
  provider cannot be asked to take it back.
- **Tests** — transition matrix over every state/event pair; a negative
  test per illegal transition; trace replay for the hold-then-release
  and suppress paths.

### Delivery attempt — `delivery`

- **Trigger** — a routing decision for exactly one channel.
- **Inputs** — notification id, channel, resolved address, redacted body
  appropriate to the sensitivity label, retry budget from the channel
  adapter.
- **Ordered steps**
  1. Persist the attempt as `pending` with the routing reason recorded.
  2. Hand the payload to the channel adapter.
  3. On success, record the provider message id and settle as
     `delivered`.
  4. On a retryable error, schedule the next attempt with backoff and
     stay `pending`.
  5. On a terminal error, or once the budget is spent, settle as
     `failed` with a reason the operator can act on.
- **Outputs** — a settled delivery row and a receipt.
- **Failure modes** — a provider 4xx is terminal (the topic or address
  is wrong, and retrying cannot fix it); a 5xx or timeout is retryable.
  Conflating the two is how delivery systems retry forever against a
  configuration mistake.
- **Retry and cancel** — exponential backoff with jitter, bounded by the
  channel's budget. A completion arriving after the parent notification
  was cancelled is discarded as stale rather than flipping a terminal
  state.
- **Tests** — retry and backoff run against an injected clock; no test
  sleeps. Adapter behavior is exercised through a fake transport; no
  test performs a real send.

### Quiet-hour hold and release — `notifications`, `routing`

- **Trigger** — a notification whose recipient has a quiet window
  covering the current local time.
- **Ordered steps** — routing returns a hold decision with a release
  time; the notification moves to `held`; the scheduler releases it when
  the window closes and re-runs routing, because preferences or channel
  availability may have changed while it waited.
- **Failure modes** — a hold that outlives its usefulness. A held
  notification older than a configurable staleness bound is released
  early or dropped with a stated reason rather than arriving as
  yesterday's news.
- **Edge cases that carry tests** — windows that cross midnight, windows
  in a timezone other than the host's, a recipient with two overlapping
  windows, and a `critical` urgency that must ignore all of them.

### Relayed delivery — `relay`, `delivery` (P1)

- **Trigger** — routing determines the selected channel cannot be served
  by this host.
- **Ordered steps** — look up nodes advertising the channel capability;
  pick one that is present and owns the recipient's device; submit a
  durable dispatch job through `vrooli-bridge`; record `node_id` and the
  correlation on the delivery row; settle when the node reports back.
- **Failure modes** — no node advertises the capability, so the channel
  is unavailable and routing falls through; the node is offline, so the
  dispatch waits rather than failing immediately (this is exactly why
  durable dispatch is used instead of the synchronous relay call); the
  dispatch times out, so the delivery fails with a reason naming the
  node.
- **Note** — the delivery row shape is identical for local and relayed
  deliveries. A reader of the timeline should not have to care which
  machine sent it, only whether it arrived.

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
| notifications / lifecycle | accepted, held, routed, delivering, delivered, failed, cancelled, suppressed | Any escape from a terminal state; `delivering` before `routed`; `delivered` with no settled delivery row; cancellation after a payload has left for a provider; suppression after routing has begun | `*.flow.json` contract, generated Quint model, transition matrix, trace replay |
| delivery / attempt | pending, delivered, failed | Settling twice; a completion applied after the parent was cancelled; a retry scheduled past the budget; `delivered` without a provider message id | `*.flow.json` contract, injected-clock retry tests, stale-completion tests |
| notifications / quiet hold | held, released, expired | Release before the window closes for non-critical urgency; a critical notification entering `held` at all; release without re-running routing | contract plus timezone and midnight-crossing matrix tests |
| relay / dispatch (P1) | selected, dispatched, acknowledged, timed_out | Dispatch to a node that advertises no matching capability; dispatch of a `secret`-sensitivity body; two dispatches for one delivery | contract with a stubbed dispatch seam |
| UI / send composer | idle, editing, submitting, submitted, failed | Submit before validation passes; stale success applied after the form was reset; double submit producing two notifications | `*.flow.json` contract, attempt-id stale completion tests |

The three that matter most are terminal-state escape, stale completion,
and double submit. Each of them produces the same user-visible bug — a
notification that appears to have arrived when it did not, or arrived
twice — which is the failure this scenario exists to prevent.

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
| Digest collapsing (OT-P1-005) | A digest window is a second timed hold running alongside quiet hours, and two independent hold mechanisms interacting is where scheduling bugs live. | Model it as a variant of the quiet hold rather than a parallel mechanism, so there is one release path, not two. |
| Acknowledgement return path (OT-P2-004) | Introduces an inbound state change on an already-terminal notification, which the current state machine forbids by design. | Model acknowledgement as a separate entity referencing the notification, not as a new notification state. Decide before OT-P2-004 starts. |
| Escalation chains (OT-P2-005) | A timer that fires a second delivery is easy to build and easy to make loop. | Requires acknowledgement first, plus an explicit maximum chain depth in the contract. |
| Self-hosted push migration (OT-P1-008) | Moving a device from a hosted topic to a self-hosted endpoint mid-flight can strand in-flight deliveries. | Treat it as re-registering the channel address rather than editing it, so in-flight deliveries settle against the old address. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
