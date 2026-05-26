# Flows — Architecture Cartographer

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
| Migration lifecycle | apply, conflicts | `arch-cart migrate start <scenario>` | Per-domain plans applied, build-green at each step, migration finalized. | Multi-stage with baseline capture, conflict resolution gate, per-domain commits, finalize transition. | Level 4 declarative contract planned for v1; Level 5 (Quint model) planned for v2. |
| Conflict resolution | conflicts | A drift finding is detected during graph-vs-manifest comparison. | Conflict transitions detected → optionally assigned/split → resolved → validated → ready for apply. | Stateful per-conflict lifecycle; resolution is reversible until apply. | Level 4 declarative contract planned for v1. |
| Auto-placement verdict | signals | Chunk needs domain assignment during graph-vs-manifest comparison. | Verdict produced (`auto_place`, `suggest`, `conflict`) with Reason + Evidence. | Pure scoring; no statefulness within a single verdict but logged for analytics. | Level 3 (matrix + replay traces against fixture chunks) for v1. |
| Per-domain apply | apply | `arch-cart apply <domain>` | File moves + import rewrites land in one atomic commit if build-green; otherwise reverted. | Stateful: baseline → plan → execute → verify → commit-or-revert. | Level 4 declarative contract; build-green guard is invariant. |
| Attachment upload (template placeholder) | notes | User/CLI uploads a file for a note. | Blob is stored and metadata is persisted. | Stateful upload request with validation and failure paths. | Level 5 workflow tests (template reference). Removed when notes domain is deleted. |

## Flow Details

### Migration lifecycle

- Owner domain: `apply` (orchestrates) with cooperation from `conflicts` and `analytics`.
- Trigger: `arch-cart migrate start <scenario>`.
- Inputs: target scenario path, optional `--mid-migration` or `--strict` flag.
- Steps:
  1. Snapshot build baseline (`go build ./...` for Go, `tsc --noEmit` for TS). Recorded in `apply_runs` as `baseline_status`.
  2. Extract graph (delegated to `go-code-graph` / `typescript-code-graph`) and cache as snapshot.
  3. Load manifest; validate; compute diff vs. graph; emit Conflict envelopes.
  4. Loop: agent runs `arch-cart conflicts list` → resolves conflicts for one domain → runs `arch-cart conflicts validate` → runs `arch-cart apply <domain>`.
  5. Per-domain apply enforces build-green guard against baseline; on regression, refuses to commit unless `--force --note`.
  6. When all required domains are clean, `arch-cart migrate finalize <scenario>` ends the migration; transitional declarations whose `expires_when` predicates fire become hard errors.
- Outputs: per-domain commits, finalized migration record, analytics history.
- Failure modes: build regression (block + `--force` option), missing scenario dependency (go-code-graph or typescript-code-graph unreachable), manifest invalid, agent abandons mid-flight (resume via `arch-cart migrate resume <scenario>`).
- Tests (planned): unit (baseline diff math, predicate evaluation); integration (full lifecycle against fixture scenarios with deliberate cycles + mislocations); regression (force-note logging).
- Requirements: OT-P0-007, OT-P0-008, OT-P1-003.

### Conflict resolution

- Owner domain: `conflicts`.
- Trigger: any detector returns a non-empty `[]Conflict` during graph comparison.
- Inputs: graph snapshot, manifest, detector registry.
- States (per conflict):
  - `detected` — surfaced by a detector; no resolution attempted yet.
  - `assigned` — agent has assigned the conflict to a target domain (optional, applies to placement-type conflicts).
  - `split` — agent has elected to split a file along chunk boundaries (applies to mixed-responsibility conflicts).
  - `resolved` — agent has applied a fix (mechanical or manual); not yet re-validated.
  - `validated` — `arch-cart conflicts validate` re-checked the graph and confirms the conflict is gone.
  - `committed` — included in a per-domain apply that landed atomically.
  - `force_resolved` — closed with `--force --note` against a still-failing validate; logged in analytics.
- Illegal transitions:
  - `detected` → `committed` (must pass through `validated`).
  - `validated` → `detected` (use `arch-cart conflicts reopen` if needed; produces a new id).
  - `committed` → anything (terminal).
- Failure modes: agent assigns to a domain that does not exist in the manifest; agent splits along chunks that re-introduce a different conflict; `arch-cart conflicts validate` times out on a very large graph.
- Tests (planned): unit (state machine completeness); integration (workbench loop against fixture conflicts); CLI smoke (`list → show → assign → resolve → validate`).
- Requirements: OT-P0-003, OT-P0-005, OT-P0-006.

### Auto-placement verdict

- Owner domain: `signals`.
- Trigger: graph-vs-manifest comparison needs a chunk's domain assignment and the chunk's path is not unambiguous.
- Inputs: immutable graph snapshot, manifest signal weights/thresholds, target chunk.
- Steps:
  1. Aggregator iterates over registered signals; each computes a `Score{Value, Reason, Evidence}` for `(chunk, candidate_domain)` pairs.
  2. Aggregator combines with manifest weights; produces final verdict per domain.
  3. Top-domain confidence compared to thresholds: `≥ auto_place` → auto-assign; `≥ suggest` → suggest with evidence; below → emit `conflict`.
  4. Verdict logged to `analytics.placements` with full signal breakdown.
- Outputs: domain assignment, or suggestion list, or conflict referencing the chunk.
- Failure modes: signal panics (each signal sandboxed; failure logged but doesn't abort verdict); no domain reaches `suggest` threshold (legitimate `conflict` state).
- Tests (planned): unit per signal; aggregator weighting tests; reproducibility tests (same graph + same weights → same verdict, byte-identical).
- Requirements: OT-P0-004.
- Related doc: [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md).

### Per-domain apply

- Owner domain: `apply`.
- Trigger: `arch-cart apply <domain>` after all that domain's conflicts are `validated`.
- Inputs: domain name, current migration record, build baseline.
- Steps:
  1. Re-validate that domain has zero unresolved conflicts.
  2. Generate operation list (file moves + import rewrites) by delegating to `go-code-graph` / `typescript-code-graph` rewrite helpers.
  3. Dry-run: stage operations; run `go build ./...` (or `tsc --noEmit`). If build broke beyond baseline, refuse unless `--force --note`.
  4. Apply: commit operations atomically; create one git commit per domain with the operation list in the body.
  5. Log to `analytics.events`.
- Failure modes: rewrite helper fails on edge syntax; build regression; partial apply (mitigated by atomic commit semantics — either all operations land or none do).
- Tests (planned): unit (operation list generation); integration (apply against fixture scenarios, verify build-green); regression (force-note path).
- Requirements: OT-P0-007, OT-P0-008.

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
| conflicts / conflict lifecycle | detected, assigned, split, resolved, validated, committed, force_resolved | detected→committed (must pass validated), validated→detected (use reopen instead), committed→anything | Planned `*.flow.json` contract + state-machine unit tests; conflict envelope's `resolved` and `resolution` fields encode terminal states. |
| apply / per-domain apply | baseline_captured, plan_generated, dry_run_ok, applied, committed, refused_build_break, force_committed | applied→baseline_captured (must commit or refuse), committed→anything | Build-green guard is a hard invariant; refused state cannot exit without `--force --note`. |
| migrate / migration lifecycle | starting, extracting, comparing, awaiting_resolution, per_domain_applying, finalized, abandoned | finalized→anything, abandoned→finalized (must `migrate resume` first) | Planned `*.flow.json` contract; finalize transition runs transitional-declaration expiry check. |
| notes / attachment upload API (placeholder) | received, bytes_stored, metadata_recorded, failed | metadata before bytes, terminal-state escape, duplicate terminal events | Template reference; removed with notes domain. |
| notes / attachment upload UI (placeholder) | idle, selected, uploading, succeeded, failed | start before select, stale completion after reset/reselect, retry without file context | Template reference; removed with notes domain. |

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
| Recipe application lifecycle | Medium — once recipes (extract-shared-types, invert-dependency, split-file) exist, each will have its own state machine for plan → preview → execute → verify → revert. | Add Level 4 contracts when the first recipe ships (OT-P1-002). |
| Cross-scenario calibration | Low — `arch-cart calibrate` (OT-P2-005) proposes weight changes; human acceptance is the only state. | Skip Level 4 modeling unless complexity emerges. |
| Maturity-scoring rollup | Low — OT-P2-006 aggregates per-scenario findings; mostly stateless query. | None planned. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
