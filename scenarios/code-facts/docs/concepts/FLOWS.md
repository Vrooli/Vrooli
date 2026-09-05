# Flows — Code Facts

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
| Describe request | facts | API/CLI/UI requests selected fact families for a bounded target. | Report returns target context, selected fact families, evidence, warnings, and cache metadata. | Cache state stores graph/report entries with deterministic hash evidence. | Unit/API/CLI/UI smoke tests plus resolver/analyzer/cache unit tests. |
| Catalog generation | catalog | Startup, watcher reconciliation, or explicit reindex starts a build. | A validated shadow generation becomes active without exposing partial state. | Shadow, active, retired, and failed generation rows. | SQLite migration, rollback, paging, dirty-worktree, and activation tests. |
| Descriptor refresh | proof | The descriptor artifact or watched manifest stamp changes. | A valid snapshot replaces the prior snapshot; an invalid refresh keeps and marks the last known-good snapshot. | Descriptor digest, generation, and reload failure evidence. | Descriptor fixture and committed-image tests. |
| Incremental reconcile | indexcontrol | A debounced file event, five-minute audit, or operator request supplies bounded changes. | Catalog, FTS, vectors, and graph projections converge without source-wide work. | Durable job cursor, progress, cancellation, outcome, and timestamps. | Coordinator batching, debounce, no-change, cancellation, and restart tests. |
| Generation promotion | indexcontrol | An operator confirms promotion of a validated shadow generation. | Qdrant alias and SQLite active pointer converge; compensation preserves the former active generation on failure. | Durable prepared, alias-promoted, committed, rolled-back, or failed promotion record. | Promotion compensation and catalog rollback tests. |

## Flow Details

### Describe request

- Owner domain: facts.
- Trigger: `DescribeCodeFacts`, `ListSurfaces`, proof RPCs, or the `/facts` workbench.
- Inputs: target kind/identifier, included fact families, optional endpoint/command/widget filters.
- Steps:
  1. Validate target shape.
  2. Normalize requested fact families.
  3. Return currently-supported target context plus typed `unsupported` evidence for later-phase families.
  4. Attach deterministic cache metadata.
- Outputs: `CodeFactsReport`, `ListSurfacesResponse`, or `ProofReport`.
- Failure modes: missing target, unsupported target kind, malformed fact-family request.
- Retry/cancel behavior: stateless in Phase 6.
- Requirements: CF-P0-004, CF-P0-005, CF-P0-008, CF-P0-009.

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| facts / describe request | validated, described, unsupported, failed | report before validation, silent unsupported family omission | handler/service tests and CLI/UI smoke tests |
| catalog / generation | shadow, active, retired, failed | empty shadow to active, retired to active, writes to retired, partial failure to active | `catalog.SQLiteRepository` transaction guards and tests |
| indexcontrol / job | queued, running, cancellation_requested, succeeded, failed, cancelled, interrupted | terminal job to running, cancellation without durable state, non-advancing cursor | `indexcontrol.Coordinator` and `SQLiteJobStore` tests |
| indexcontrol / promotion | prepared, alias_promoted, committed, rolled_back, failed | commit before alias, active pointer after failed alias, split generation left unrecorded | durable promotion ledger and compensation tests |

### Catalog generation

1. Resolve governed repository roots.
2. Stream tracked and non-ignored files from Git. Use the filesystem iterator for an external root.
3. Skip deleted tracked paths, symlinked directories, ignored trees, vendor trees, build output, and transient protobuf verification trees.
4. Hash and classify one regular file at a time.
5. Write bounded batches to the shadow generation.
6. Record the source and descriptor digests.
7. Validate that the shadow generation is complete.
8. Retire the former active generation and promote the shadow generation in one transaction.

Cancellation or a discovery/write failure marks the shadow generation failed. It does not change the active generation.

### Incremental reconciliation and control

1. Coalesce repeated path events for 500 ms and cap event delay at 10 seconds.
2. Deduplicate by path and retain the latest operation.
3. Read at most 256 changes by durable cursor unless a lower configured limit applies.
4. Apply one bounded catalog, lexical, vector, and graph batch.
5. Persist cursor, progress, cancellation, error, and outcome after every batch.
6. Mark a running job `interrupted` after restart; resume or explicitly replace it from its durable cursor.
7. Run a manifest audit every five minutes so a missed watcher event repairs.
8. Validate a shadow generation before any alias or catalog promotion.
9. Record promotion intent, move the vector alias, and then activate SQLite. If SQLite activation fails, restore the former alias and record `rolled_back`.
10. Keep the previous complete generation until confirmed rollback or cleanup.

Typed Connect RPCs and the `code-facts index` CLI group expose status,
reconcile, reindex, cancel, promote, rollback, and cleanup. Reindex, promote,
rollback, and mutating cleanup require explicit confirmation. Cleanup supports
a non-mutating preview.

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
| None yet. | Generated scaffold. | Add real scenario workflows when domains have stateful behavior. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
