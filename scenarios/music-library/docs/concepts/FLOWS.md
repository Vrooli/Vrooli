# Flows — Music Library

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
| Library scan | library | Source root added, or rescan | Track index updated | Incremental and resumable; distinguishes moved, missing, and deleted | Level 4 |
| Decomposition batch | decomposition | New or changed tracks | Attributes cached locally | Resumable, per-track partial, tolerates upstream unavailability | Level 4 |
| Playback session | playback | Listener starts playback | Audio output plus interaction events | Queue and transport states, gapless boundaries, transcode cache miss | Level 5 |
| Comparison round | elicitation | Listener opens the compare surface | Comparisons recorded, profile refit | Informative pair selection, undo, ordering effects | Level 4 |
| Preference refit | preference | New comparison, new signal, or reset | Updated profile | Background; never blocks playback; versioned and reversible | Level 4 |
| Ranking with explanation | ranking | The queue needs a next track | Ranked track plus its reason | Single pass; exploration and exploitation distinguished | Level 4 |
| Generation loop | generation | Background schedule or directed request | Candidate tracks | Queue, budget eviction, retention on confirmation | Level 5 |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

### Playback session — `playback`

- Trigger: listener starts playback.
- Steps: resolve the next track through `ranking`, ensure a playable rendition
  (cache hit, or transcode), start transport, emit interaction events, prepare the
  following track before the current one ends.
- Illegal transitions: an audible gap at a gapless boundary; a transport command
  that does not take effect within the interaction budget.
- Failure modes: transcode cache miss on an uncommon format (must stay within the
  transport budget); source file missing (mark missing, skip, do not delete).
- Why Level 5: correctness here is judged by the listener's ear, and the
  next-track path overlaps live playback.

### Ranking with explanation — `ranking`

- Trigger: the queue needs a next track.
- Steps: retrieve a shortlist from the vector index, score it against the active
  context profile, apply calibration, apply the exploration policy, emit the chosen
  track **with the reason that produced it**.
- Invariants: the explanation is produced by the same pass that produced the
  ranking — never reconstructed afterwards; a track chosen for exploration is
  labelled as such rather than presented as a confident recommendation.
- Blindness rule: this flow cannot observe offer state. `offers` decorates the
  finished ranking and may not reorder or filter it.
- Failure modes: profile unfitted (say so plainly and route to comparison rather
  than presenting arbitrary output as recommendation).

### Comparison round — `elicitation`

- Trigger: listener opens the compare surface.
- Steps: select the pair that most reduces model uncertainty, present, record the
  judgment, trigger a background refit.
- Statefulness: undo is supported; deleting a comparison refits rather than leaving
  a stale profile.
- Known confound: presentation order affects choice, so ordering is randomised and
  recorded.

### Preference refit — `preference`

- Trigger: new comparison, accumulated implicit signal, or an explicit reset.
- Statefulness: background, incremental where possible, versioned so a reset is a
  refit rather than an amnesia — raw history is retained.
- Hard rule: a refit never blocks playback.
- Compatibility: a profile records the embedding model it was fitted against. If
  that model changes upstream, the profile is marked stale and requires an explicit
  refit rather than being read against a different coordinate space.

### Library scan — `library`

- Trigger: source root added, or a rescan.
- Steps: walk the root, derive content identity, reconcile against the index,
  extract embedded metadata, enqueue changed tracks for decomposition.
- Invariant: **source files are opened read-only.** No path in this flow writes to
  a source root.
- Statefulness: incremental and resumable; a track whose path disappears is marked
  missing and re-links by content when it returns, preserving ratings and history.

### Decomposition batch — `decomposition`

- Trigger: tracks entering the index or their attributes going stale.
- Statefulness: resumable across restarts; per-track partial results are recorded as
  partial rather than complete.
- Failure modes: `music-tools` unavailable (queue and report; playback and browsing
  are unaffected because attributes are cached).
- Reality: for a large library this is an overnight-or-longer job. Progress
  reporting must be truthful about that rather than implying imminent completion.

### Generation loop — `generation`

- Trigger: background schedule, or a directed listener request.
- Steps: sample target regions from the profile under the exploration policy, submit
  composition to `music-tools`, decompose the result through the same path as owned
  audio, admit it to the candidate pool.
- Statefulness: budgeted; unconfirmed candidates evicted oldest-first.
- Invariant: generated audio is **always disclosed as generated**, in the interface
  and in exported metadata.
- Blindness rule: like `ranking`, this flow cannot observe offer state.

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
| playback / session | `idle` → `loading` → `playing` ⇄ `paused` → `ended`; `loading` → `unplayable` | An audible gap at a gapless boundary; `playing` without a resolved rendition | `*.flow.json` contract, replay tests, and audio-level assertion of the boundary |
| decomposition / batch | `pending` → `running` → `partial` \| `complete`; `running` → `blocked` when upstream is down | Reporting `complete` when any layer failed; losing progress across restart | Contract plus resumability replay tests |
| preference / refit | `fitted` → `stale` → `refitting` → `fitted`; `unfitted` initial | Ranking as if `fitted` while `unfitted` or `stale`; refitting on the playback path | Contract test; a stale profile must surface as such in ranking output |
| elicitation / comparison | `pair-selected` → `shown` → `judged` \| `skipped` → `recorded` | Recording a judgment without a presented pair; reusing a pair within a round | Contract tests including undo and ordering randomisation |
| generation / candidate | `queued` → `generating` → `decomposing` → `available` → `confirmed` \| `evicted` | `available` before decomposition; evicting a `confirmed` candidate; surfacing a candidate without its generated-content disclosure | Contract tests; disclosure asserted at the surface, not only at the record |
| library / scan | `idle` → `scanning` → `reconciling` → `idle` | Any write to a source root; deleting a track whose file is merely missing | Read-only open enforced and tested; missing-versus-deleted asserted |

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

- **Section-level skip attribution** — designed and reliability-gated, but held as a
  hypothesis with no prior art. It is not modelled as a load-bearing flow because
  nothing depends on it. See [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
- **Multi-listener profiles** — the first deployment is single-listener. Scoping and
  isolation must be specified before any shared deployment.
- **Offline synchronisation** — real work inherited from owning the player, but no
  ordered-state model until the delivery surface is chosen.
- **Adapter-based personalisation** — blocked on hardware; personalisation today is
  a light head over frozen embeddings.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
