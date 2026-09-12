# Flows — Audio Tools

> Plan: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`

## Headline flows

### Transcribe (STT chain)

The existing API/CLI/consumer transports enter Audio Tools' shared STT chain
and session/segmenter machinery. Speech tier order is configurable (currently
local-first); a tier flag does not instantiate a provider. See
[INTEGRATIONS.md](INTEGRATIONS.md) for actual wiring: local/BYOK adapters exist,
but the Vrooli provider is not supplied by the inspected bootstrap.

Target flow for PRD OT-P0-004 through OT-P0-010:

```text
consumer voice action and prior route/data/spend policy
  -> bounded capability selection and explicit preparation state
  -> microphone/capture + session admission
  -> local or BYOK adapter
     OR shared owned-service authorization/reservation -> inference adapter
  -> revisable partials + stable committed intervals
  -> stop/cancel/fault -> final drain or explicit retained/recoverable outcome
  -> owned route only: shared delivery/settlement reconciliation
  -> owner-backed outcome receipt and metadata-only diagnostics
```

Route selection, permission, capture and provider readiness are separate events;
capture may buffer only within its declared limits. Batch-only adapters expose
that limitation rather than fabricating incremental text. A provider failure
may enter another route only under previously authorized policy. Do not infer
subscription consent from a lower-tier fallback or a post-response notification.
Owned-service arrows above are target behavior, not a completed gateway.

### Synthesize (TTS chain)

Same shape as Transcribe but per-call resolves canonical voice via the catalog:
```
voice canonical id → voice_overrides["tier:provider-id"] (if any) → adapter mapping → adapter default + warning
```

### Summarize (Summarize chain)

Same policy-permitted tiering concept, with capability-specific ordering. Local summarization uses `internal/summarize` (Ollama); BYOK uses its OpenRouter adapter. Hosted summarization remains unimplemented in the inspected LPBS client.

### Browser-voice WS session

```
client opens WS /api/v1/voice/stream?voice=…&language=…
  └─ handlers/stt/stream_ws.StreamWSHandler.ServeHTTP
       ├─ session.New(transport="browser-voice", voice, language, CancelHook)
       ├─ session.Registry.Add
       ├─ stt.Segmenter (transport-agnostic streaming pipeline)
       └─ on disconnect: session.Close("ws-disconnect"); registry.Remove
```

Concurrent `SessionService.Subscribe` observers attach to the same `session.Session` and receive every transcript/VAD/barge-in event.

### Barge-in

```text
transport VAD reports speech_start during in-flight TTS
  └─ session.BargeIn(BargeInVAD)
       ├─ clear inflight
       ├─ CancelHook(VAD, eventID)  ── transport stops streaming TTS bytes
       └─ EmitEvent(BargeInCancel)  ── observers see the cancel
```

Historical design candidate: p95 ≤ 100 ms from VAD event to observer notification. This is not an adopted portable-voice-v1 SLO or OT-P0-008's meaning; review a separate TTS/barge-in acceptance band under OT-P1-004.

### Web-console adoption call

```text
web-console handler / orchestration
  └─ audioports.RemoteSpeechToText.Transcribe
       └─ integrations/audiotools.Client.STT.Transcribe (Connect-RPC)
            └─ on transport failure: Client.HandleTransportFailure()
                 └─ resolver.Invalidate() → next call re-resolves URL
```

### Usage reporting and owned settlement

Current handler usage recording feeds bounded asynchronous local history.
It can drop rows under pressure and is not the authoritative customer ledger.
The LPBS remote reporter is a placeholder; its existence does not establish a
durable remote write. See [Usage reporting](../domains/usage/reporting.md).

The owned-service target requires shared authorization before billable work,
stable operation/session/interval identity, durable delivery evidence, idempotent
settlement and recovery of stranded reservations after cancellation or restart.
The policy and simulation cases live in [MONETIZATION.md](../business/MONETIZATION.md).
A missing receipt is pending/unknown, not a zero charge or proof of free service.

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
| Attachment upload | notes | User/CLI uploads a file for a note. | Blob is stored and metadata is persisted. | Stateful upload request with validation and failure paths. | Level 5 workflow tests: matrix, traces, declarative spec, checked Quint model, generated artifacts, and production replay. |

## Flow Details

### Attachment upload

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

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| notes / attachment upload API | received, bytes_stored, metadata_recorded, failed | metadata before bytes, terminal-state escape, duplicate terminal events | `*.flow.json` contract, generated Quint model, generated formal artifact replay, side-effect cleanup tests |
| notes / attachment upload UI | idle, selected, uploading, succeeded, failed | start before select, stale completion after reset/reselect, retry without file context | `*.flow.json` contract, generated Quint model, generated formal artifact replay, attempt-id stale completion tests |

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
flow-verifier flows new "api/internal/<domain>" --flow-id "<flow-id>" --lang go --root .
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
