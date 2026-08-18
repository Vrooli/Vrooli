# Flows — Document Manager

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

Flows marked **P2 flow** belong to the write spine (`templates`,
`composition`, `render`). They are modeled here so the boundary is
designed, and are **not scaffolded** until the read spine ships — see the
write-spine rows in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Document ingestion | intake → derivation → anchors | Upload, watched-directory event, or connector fetch. | A content-addressed document with a versioned derivation and anchored units. | Long-running, resumable, per-stage retry; dedupe makes replay a no-op. | Target level 4 — declarative contract. |
| Tier routing | derivation | A document needs a derivation and its privacy class is resolved. | Exactly one tier executes and records why it was chosen. | Stateless decision, stateful escalation on low confidence. | Target level 4. |
| Sensitivity resolution | sensitivity | A document enters the pipeline, before tier selection. | A privacy class and the routing profile it selects. | Stateful when detection needs inference; fails closed. | Target level 5 — the residency guarantee is the product claim. |
| Redaction review | sensitivity | Detections exist and an operator opens review. | A redaction manifest and a redacted derivative. | Stateful, human-gated; proposals never auto-apply; stale confirmation must be rejected. | Target level 5 — defensibility depends on it. |
| Enrichment | enrichment | Units exist and enrichment is requested or scheduled. | Summaries, extractions, embeddings with full metadata. | Per-unit retry, partial success, credit reservation on the metered path. | Target level 4. |
| Re-derivation | derivation → anchors → retrieval | Parser or model upgrade, or an operator correction in the Reader. | A new derivation version; geometric anchors resolve unchanged, logical anchors gain an alignment; the retrieval index is rebuilt. | Stateful, bulk, cancellable; must never mutate a prior version. | Target level 4. |
| Corpus retrieval | retrieval | A caller queries this corpus, directly or through search-hub. | Ranked anchored units, filtered by collection and privacy class. | Stateless read over a rebuildable index. | Target level 4 — privacy filtering is a correctness property, not a convenience. |
| Ledger publication | handoff | A consumer records a finding and a scope binding exists. | One scoped, idempotent ledger entry carrying an anchor as provenance — never unit text. | Queued with retry when the ledger is unreachable; cursor-based. **Optional flow: the scenario is fully useful without it.** | Target level 4. |
| Attestation export | custody | Operator requests a receipt or attestation. | A deterministic, human-readable residency report. | Stateless read over append-only records. | Target level 3. |
| Metered execution | enrichment, derivation (T3) | A paid-tier operation is requested. | Work performed and credits settled to actual usage. | Reserve → execute → finalize; must refund on under-use and never partially deliver. | Target level 5 — money moves. |
| Document generation | composition → render → intake | A caller renders a spec version under a template version to a target. | A render version, rendered bytes, a block alignment, and the artifact ingested back as an ordinary corpus document. | Long-running for large targets; append-only versions make replay a no-op. **P2 flow.** | Target level 4. |
| Template switch | templates → render | An operator or agent re-renders an existing spec under a different template. | A new render version under the new template, plus a per-element report of what the target could not represent. | Stateful, previewable, must not commit before the unrepresentable report is shown. **P2 flow.** | Target level 5 — a silent drop here is the write-side equivalent of an anchor that lies. |
| Source refresh | composition | Bindings are re-run, on demand or because a source moved. | A **new spec version** with re-resolved values, each snapshotted with its timestamp. | Per-binding retry and partial success; never mutates a prior spec version. **P2 flow.** | Target level 4 — this is the only verb that can change what the document claims. |
| Agent-driven spec edit | composition (via the Composer chat) | A chat turn requests a change to a document. | A spec mutation, a new spec version, and a custody event naming actor, verb and whether inference was involved. | Stateful per turn; stale completion after a concurrent edit must be rejected; undo is revert-to-version. **P2 flow.** | Target level 5 — parity and custody both depend on it. |
| Template proposal review | templates | A chat turn or operator proposes a template edit. | A confirmed template version, or a rejection; either way an audited decision. | Human-gated; proposals never auto-apply, exactly like redaction review. **P2 flow.** | Target level 5 — blast radius is every document using the template. |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| intake / ingestion | received, addressed, typed, deriving, derived, anchored, failed | typed before addressed; anchored before derived; terminal-state escape; duplicate terminal events for one content address | `*.flow.json` contract, generated Quint model, replay tests, dedupe idempotency tests |
| sensitivity / resolution | pending, detecting, classified, profile_bound, failed_closed | **tier selection before profile_bound**; downgrade of a class without an audited actor; any transition from failed_closed to a remote route | `*.flow.json` contract, generated Quint model, replay tests, an explicit test asserting a confidential document fails closed |
| derivation / tier routing | eligible, tier_selected, parsing, parsed, escalated, failed | selecting T3 while privacy class is confidential or secret; escalation without a recorded reason; parsed without a version | `*.flow.json` contract, generated Quint model, tier-selection table tests |
| derivation / re-derivation | requested, running, superseded, cancelled, failed | mutating a prior version; superseding before the new version is durable; resolving an old anchor to a new region | `*.flow.json` contract, cross-version anchor resolution tests |
| sensitivity / redaction review | proposed, under_review, confirmed, applied, rejected, stale | applied before confirmed; confirmation of a proposal superseded by a newer detection pass; auto-apply on any path | `*.flow.json` contract, generated Quint model, stale-confirmation tests |
| enrichment / metered execution | reserved, executing, finalized, refunded, insufficient_credits, failed | executing without a reservation; finalizing twice; delivering output after a failed reservation; charging a BYOK caller | `*.flow.json` contract, generated Quint model, credit-settlement replay tests |
| handoff / publication | pending, publishing, published, retrying, unbound | publishing without a scope binding; duplicate entries for one idempotency key; publishing unredacted units from a redacted document | `*.flow.json` contract, idempotency tests |
| UI / Reader correction | idle, unit_selected, editing, submitting, versioned, failed | submitting without a selection; stale completion after the user reselects; overwriting rather than versioning | `*.flow.json` contract, attempt-id stale completion tests |
| render / production (P2) | requested, routing, rendering, rendered, partial, failed | rendering before a template version is pinned; emitting a render version without a block alignment for a target whose renderer declares support; reaching `rendered` while any element was unrepresentable — that is `partial`, and it must say which | `*.flow.json` contract, generated Quint model, terminal-state coverage tests, round-trip fidelity gate |
| templates / switch (P2) | previewing, unrepresentable_reported, confirmed, re-rendering, committed, cancelled | committing before the unrepresentable report has been produced; dropping an element without reporting it; discarding any layer other than bindings | `*.flow.json` contract, generated Quint model, an explicit test asserting no silent element loss |
| templates / proposal review (P2) | proposed, under_review, confirmed, applied, rejected, stale | applied before confirmed; confirming a proposal superseded by a newer template version; auto-apply on any path; a per-document override entering this flow at all | `*.flow.json` contract, generated Quint model, stale-confirmation tests mirroring redaction review |
| composition / spec mutation (P2) | received, validating, versioned, rejected, failed | mutating rendered bytes rather than the spec; versioning without validation; a `render` minting a spec version; losing a prior version on revert | `*.flow.json` contract, generated Quint model, an AST assertion that no verb writes rendered bytes, revert-as-undo tests |
| UI / Composer chat turn (P2) | idle, composing, awaiting_confirmation, applying, applied, failed | applying a corpus-scoped action without a separately authorized scope binding; applying a template edit without confirmation; stale completion after a concurrent spec edit; constructing anything but a generated client | `*.flow.json` contract, attempt-id stale completion tests, the `DOC-P2-021` AST check |

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
| Tier-3 vision parse | Cannot be modeled end-to-end until AI Gateway declares a vision role. Modeling it against a hypothetical role would encode an interface that may not match what ships. | Model after `vision.default` lands; blocks `DOC-P1-001`. |
| Corpus-scale re-derivation | The single-document re-derivation flow is modeled above; the bulk sweep adds cancellation, partial failure, and progress semantics over thousands of documents. Getting it wrong risks a long-running job that cannot be stopped. | Model alongside `DOC-P1-003`, reusing the single-document contract as the inner step. |
| Legal hold interaction with prune | Hold suppresses every prune path, so the interaction is a cross-cutting invariant rather than a flow of its own. Under-modeling it risks deleting data under hold. | Encode as an invariant in the corpus retention contract at `DOC-P1-015`, not as a separate state machine. |
| Embedding retarget migration | Changing embedding role, model or dimension invalidates stored vectors. Rare, high-blast-radius, and partly owned by AI Gateway policy. | Model when the first retarget is actually planned; record the decision in `../internal/DECISIONS.md` first. |
| Corpus-wide re-render on a template edit | The single-document switch is modeled above; the bulk sweep across every document using a template adds cancellation, partial failure and progress over potentially thousands of documents — and unlike bulk re-derivation it changes what people *see*, so a half-applied sweep is visible rather than merely incomplete. | Model alongside `DOC-P2-014`, reusing the single-document switch contract as the inner step, exactly as bulk re-derivation reuses the single-document one. |
| Corpus-scoped agent action | "Restyle every deck in this collection" needs a separately authorized scope binding (`DOC-P2-025`) and inherits the bulk-sweep concerns above. Modelling it before per-collection access control (`DOC-P1-014`) exists would encode an authorization shape that may not match what ships. | Model after `DOC-P1-014`. Until then the agent is document-scoped and the flow does not exist. |
| Spec schema migration | A spec is the write spine's authority, so an existing spec must keep rendering across a schema change. Rare and high-blast-radius, like the embedding retarget. | Model when the first schema change is actually planned; record the decision in `../internal/DECISIONS.md` first, per the additive rule in `DATA.md`. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
