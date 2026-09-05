# Flows — Workflow Health

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

Workflow Health is a workflow governance scenario, so the core product
behavior is stateful even before individual formal flow models are added.
The first implementation pass should inventory these flows, then promote
the highest-risk execution and fix flows to formal `flow.json` models.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Catalog scan | catalog | Operator, provider validation, search refresh, or UI scenario selection. | Deterministic catalog snapshot with asset facts and dependency edges. | Stateless read with snapshot versioning and stale registry findings. | Level 1 now; scanner golden tests planned. |
| Scenario validation | validation | Test Genie or workflow-health CLI/API calls `ValidateScenario`. | Shared validation response with findings, metrics, maturity, native detail, and optional execution results. | Request-scoped run with optional execution branch and fix recommendations. | Level 2 target before provider migration. |
| Workflow execution | execution | Validation includes execution or operator explicitly runs a selected case/flow. | BAS run evidence and artifact references. | Safety preflight, BAS call, artifact ingestion, terminal status. Lease/claim and stale completion guards remain migration follow-up work. | Level 2 implemented; Level 4 target before Test Genie migration. |
| Search indexing | search | Catalog scan, Search Hub provider query, or operator request. | Typed workflow leaves for workflow-health and Search Hub. | Rebuildable projection with ranking and safety metadata. | Implemented for local search and workflow-health self-registration. |
| Fix preview/apply | remediation | CLI/API/UI requests deterministic remediation. | Diff-like plan or applied mechanical repair. | Preview version, confirmation/apply, idempotence, stale preview refusal. | Level 4 target before broad autofix use. |
| Operator UI review | UI | Operator opens workflow-health or switches the selected scenario. | Dense overview, inventory, search, runs, findings, fixes, and settings surfaces for the current workflow posture. | Client-side route and filter state backed by deterministic catalog/search/validation shapes. | Level 1 implemented; runtime API-backed UI flows and BAS captures remain hardening follow-up. |
| Test Genie migration | validation | Provider contract passes and catalog is updated. | `workflow` phase delegates to workflow-health; `playbooks` aliases workflow. | Migration state from native runner to delegated provider with deprecation path. | Level 2 target with catalog guard tests. |

## Flow Details

### Catalog scan

- Owner domain: catalog.
- Inputs: target scenario root, `bas/registry.json`, `bas/cases`, `bas/flows`, `bas/actions`, `bas/seeds`, requirements registry, selector registry, and workflow JSON files.
- Steps: resolve scenario root, parse registry, discover workflow files, parse assets with structured JSON, extract metadata/refs/dependencies, classify role, sort deterministically, emit snapshot and findings.
- Failure modes: malformed JSON, unsupported legacy shape, missing registry, stale registry entry, duplicate stable ID, unresolved subflow, unresolved requirement link.
- Retry/cancel behavior: caller can retry after file changes; scan should be deterministic for the same tree.
- Requirements: `REQ-P0-001`, `REQ-P0-002`.

### Scenario validation

- Owner domain: validation.
- Inputs: shared validation request, target scenario, static/execution options, selected assets, fix mode.
- Steps: scan catalog, evaluate rules, compute severity-gated status, build maturity assessment, optionally call execution, pack native detail, return shared response.
- Failure modes: scan errors, safety blockers, provider timeout, BAS unavailable, serialization errors.
- Retry/cancel behavior: Test Genie owns run cancellation; workflow-health must make each validation request idempotent and timeout-aware.
- Requirements: `REQ-P0-003`, `REQ-P2-001`.

### Workflow execution

- Owner domain: execution.
- Inputs: selected case/flow assets, safety profile, seed/reset config, execution mode, routed isolation proof, BAS options.
- Steps: run safety preflight, decode workflow JSON, validate resolved workflow through BAS, execute selected asset through BAS, fetch timeline, write deterministic run summary and artifact references.
- Failure modes: unsafe mutation, missing route isolation, BAS unavailable, navigation failure, selector failure, timeout, stale completion after cancellation.
- Retry/cancel behavior: current service is request-scoped and preserves artifact evidence for completed BAS attempts; claim/lease and stale completion protections are still required before Test Genie migration.
- Requirements: `REQ-P0-005`, `REQ-P0-006`.

### Search indexing

- Owner domain: search.
- Inputs: latest catalog snapshot, validation summaries, run signals, safety metadata.
- Steps: produce typed leaves, shape result text/metadata, rank by intent and safety, expose provider/API/CLI result sets.
- Failure modes: stale catalog, unsafe flow missing guardrails, accidental fragment promotion.
- Retry/cancel behavior: rebuild projection from source catalog; no mutation.
- Requirements: `REQ-P1-001`.

### Fix preview/apply

- Owner domain: remediation.
- Inputs: target scenario, latest catalog snapshot, selected finding IDs/rules, apply flag.
- Steps: compute deterministic diff plan, reject ambiguous or behavioral edits, persist preview, apply only against the expected catalog version, rescan.
- Failure modes: stale preview, conflicting file changes, ambiguous requirement label, non-mechanical workflow behavior change.
- Retry/cancel behavior: preview is repeatable; apply must be idempotent or fail without partial hidden state.
- Requirements: `REQ-P1-002`.

### Operator UI review

- Owner domain: UI composition over catalog, validation, search, execution, and remediation.
- Inputs: current scenario selection, catalog asset summaries, search leaves, latest run timeline, stable findings, and fix-preview records.
- Steps: render overview maturity and asset posture, inspect inventory tables, rank workflow search results with safety chips, review run timeline/artifacts, inspect finding remediation, and preview deterministic fixes.
- Failure modes: stale sample data before API-backed hooks, missing responsive state, duplicate landmarks, untranslated UI copy.
- Retry/cancel behavior: client-side only in the current slice; API-backed refresh/retry behavior belongs with the relevant domain hook.
- Requirements: `REQ-P1-003`, `REQ-P5-001`.

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| execution / workflow run | static_only, preflight, refused, dry_run, running, writing_artifacts, passed, failed | BAS call before safety pass, artifact write before a BAS execution attempt, mutating run without confirmation and routed isolation proof | current execution service tests; planned `flow.json`, lease tests, fake BAS stale-completion tests |
| remediation / fix preview | draft, previewed, applied, stale, rejected | apply before preview, apply after catalog version changed, silent behavioral edit | planned `flow.json`, temp-fixture idempotence tests |

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
