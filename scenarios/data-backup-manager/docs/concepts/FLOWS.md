# Flows — Data Backup Manager

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

These are the intended product flows for the locked design. They are
not modeled yet; the rows below name them, their owner domain, and the
states they will need. The flow modeling machinery (Levels 2–5,
described later in this document) is the target maturity, not the
current state.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Target self-registration | targets | A scenario calls register at its lifecycle. | Target upserted (owner+name), or no-op if unchanged. | Idempotent upsert; create-vs-update branch. | Implemented: service-level idempotency and reconstruction tests. |
| Default coverage acceptance | coverage | Operator runs `coverage accept-defaults` (or the UI "Register recommended"). | All non-sensitive discovered durable targets registered; sensitive ones skipped unless opted in. | Idempotent bulk upsert; per-item success/skip/fail. | Coverage service tests (split, dry-run, idempotency, partial failure). |
| Scheduled backup run | runs | In-process scheduler fires a plan, or operator/scenario triggers on-demand. | One snapshot per target per destination, run + per-target outcomes recorded. | Stateful job with per-target fan-out and partial failure. | Planned: Level 5 flow model (states, traces, checked model, replay). |
| Verified restore | restores | Operator/scenario requests restore or verify of a target. | Target restored to a location, or test-restored to scratch + checksummed (last-verified recorded). | Stateful job with verify gate. | Planned: Level 5 flow model. |
| Generic snapshot audit | audits | Operator/scenario requests an audit of a snapshot. | Snapshot restored to scratch (recoverability), live target captured to scratch (read-only), both walked and compared by generic inventory; proof persisted. | Stateful async job; pass/diff/drift/failed terminal. | Implemented: service + walker + dbcheck + comparator tests (`api/internal/audits/*_test.go`). |
| Storage-limit block | destinations / runs | A run would write past a destination's cap. | Run is blocked, an alert is raised, no bytes are written; never silent eviction. | Branch on cap-check before write. | Implemented: destination service cap-policy and usage tests. |

## Flow Details

### Target self-registration

- Owner domain: targets.
- Trigger: an owning scenario calls the register operation at its
  lifecycle (via the CLI), mirroring agent-manager's `EnsureProfile`.
- Inputs: owner, name, source kind, locator, optional quiesce-hook
  references (P1).
- Steps:
  1. Validate owner + name + source kind + locator.
  2. Look up the existing target by `owner + name`.
  3. Upsert: create if absent, update locator/kind if changed, no-op if
     identical.
  4. Return the (idempotent) target record.
- Outputs: the target record, or a typed validation error.
- Idempotency: re-registration on boot is the normal path. The catalog
  is a cache; the registration model is reconstructable.
- Requirements: OT-P0-001.

### Default coverage acceptance

- Owner domain: coverage (composes discovery + targets + plans + runs +
  restores; owns no scanner logic of its own).
- Trigger: operator runs `coverage accept-defaults`, or clicks "Register
  recommended" in the UI coverage banner.
- Inputs: `include_sensitive` (default false), `dry_run` (default false).
- Steps:
  1. List discovery target suggestions (already filtered to unregistered +
     non-dismissed).
  2. Partition into non-sensitive (default recommendations) and sensitive
     (review-only).
  3. For each non-sensitive suggestion — and sensitive ones only when
     `include_sensitive` — register it via the targets service (idempotent
     upsert, locators only, no content read). Under `dry_run`, register nothing
     and report what would be registered.
  4. Return per-item `accepted` / `skipped_sensitive` / `failed`.
- Outputs: the accept result; sensitive suggestions stay review-only unless
  explicitly opted in.
- Guard: `plans create` / `plans update` block with `failed_precondition` while
  non-sensitive recommendations remain unregistered, unless
  `allow_incomplete_coverage` is set.
- Idempotency: discovery excludes registered targets, so re-running accepts
  nothing new.

### Scheduled backup run

- Owner domain: runs (capture via sources, artifacts via kopia).
- Trigger: in-process scheduler fires a due plan, or an operator/
  scenario triggers a run on demand.
- Inputs: plan (its member targets and destinations, schedule,
  retention).
- Steps:
  1. Open the run; record plan, trigger source, start time.
  2. For each member target × destination:
     a. (P1) run the pre-quiesce hook if declared.
     b. Capture the source via its source-kind handler (filesystem tar,
        SQLite `VACUUM INTO`, `pg_dump`, Redis prefix dump, Qdrant
        snapshot, or object-storage mirror), while source resource CLIs own
        their credential resolution.
     c. Check the destination cap; if the write would exceed it, block
        (see Storage-limit block) and mark the target failed.
     d. `resource-kopia snapshot create` into the destination
        repository; record the snapshot reference.
     e. Apply the plan's retention via kopia policy.
     f. (P1) run the post-quiesce hook.
  3. Record per-target outcomes and update last-success-per-target.
  4. Close the run (success / partial-failure / failure) and emit a
     backup-outcome event for platform monitoring.
- Outputs: a run record with per-target outcomes; events.
- Failure modes: source resource unreachable, credential authority unavailable
  (fail closed — never run unencrypted), kopia unreachable, cap
  exceeded. A single target's failure does not abort the others; the
  run is partial-failed.
- Requirements: OT-P0-002, OT-P0-005, OT-P0-009, OT-P0-010.

### Verified restore

- Owner domain: restores.
- Trigger: operator/scenario requests `restore` (to a chosen location)
  or `verify` (test-restore to scratch).
- Inputs: target, snapshot selection, destination, restore mode
  (restore | verify), restore location (restore mode) or scratch
  (verify mode).
- Steps:
  1. Resolve the snapshot in the destination repository.
  2. `resource-kopia snapshot restore` to the location (restore) or to
     a scratch directory (verify).
  3. In verify mode, checksum the restored artifact and compare against
     the expected manifest.
  4. Record the restore/verify outcome; in verify mode update
     last-verified-per-target.
- Outputs: a restore record; in verify mode, a pass/fail with checksum
  evidence.
- Gate property: a verified restore is the precondition for removing a
  target's committed runtime data from git. No verify, no git removal.
- Requirements: OT-P0-006; OT-P1-004 (granularity) extends step 1.

### Generic snapshot audit

- Owner domain: audits.
- Trigger: operator/scenario requests `audits run` for a target +
  destination + snapshot.
- Inputs: target, destination, snapshot, and two cost knobs —
  `include_content_hash` and `include_sqlite_checks` (clients default
  both on; opt out for huge trees).
- Steps (async on a background worker; the request returns a `requested`
  record and the CLI/UI poll to terminal):
  1. Restore the snapshot to a scratch directory (`resource-kopia
     snapshot restore`). Success sets `restorable=true`.
  2. Resolve the snapshot's start time from the engine snapshot list (for
     drift interpretation).
  3. Walk the restored tree with one generic filesystem walker → snapshot
     inventory (counts, bytes, path-list hash, optional content hash,
     SQLite candidates).
  4. Capture the live target to a *second* scratch directory via the
     same `sources.Capturer` used by backup — read-only on live, never
     mutating it.
  5. Walk the captured live tree → live inventory.
  6. For each discovered SQLite file (detected by magic header, not
     extension), run a read-only `PRAGMA integrity_check`, page facts, and
     a normalized schema hash.
  7. Compare the two inventories by generic signals only and persist the
     completed audit. Both scratch directories are always removed.
- Outputs: an audit record with the verdict — `matches`, the specific
  generic `mismatches`, and `live_newer_than_snapshot` (a mismatch
  explained as drift when live changed after the snapshot timestamp).
- Genericity property: DBM never encodes any scenario's domain objects.
  The proof is counts, bytes, hashes, and SQLite integrity — never file
  contents or secrets. Scenario-specific semantic proof, if ever needed,
  belongs to future scenario-owned hooks, not DBM core.
- Requirements: DBM-RST-003 (OT-P0-006).

### Storage-limit block

- Owner domain: destinations (enforced inside a run).
- Trigger: a run's write to a destination would push usage past the
  configured cap.
- Inputs: destination cap, current usage from kopia repository stats,
  the pending write.
- Steps:
  1. Read current usage (kopia repo stats) before writing.
  2. If `usage + pending > cap`: raise an alert, refuse the write, mark
     the target failed for this run. **No eviction occurs.**
  3. Otherwise proceed with the snapshot write.
- Outputs: either a normal write or a blocked-with-alert outcome.
- Locked default: **alert + block**. Reclaiming space is only ever done
  through explicit retention on a plan, never by the cap check.
- Requirements: OT-P0-008.

## State Machines

These are the target state sets for the modeled flows once they reach
Level 2+. They are design intent, not implemented machines.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| runs / backup run | pending, capturing, snapshotting, completed, partial_failed, failed | snapshotting before capturing, terminal-state escape, success after a recorded target failure | Planned: `*.flow.json` contract + generated checked model + replay tests |
| restores / verified restore | requested, restoring, verifying, verified, restored, failed | verifying without a completed restore-to-scratch, marking verified after a checksum mismatch | Planned: `*.flow.json` contract + generated checked model + replay tests |

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
| Scheduled backup run | High — partial failures, cap blocks, and per-target fan-out are easy to get subtly wrong; a silently-skipped target undermines the recovery story. | Model at Level 5 (states above, named traces for success / partial-failure / cap-block) once the runs domain has executable transition logic. |
| Verified restore | High — the verify gate is the safety contract for removing data from git; a false "verified" is the worst failure. | Model at Level 5 with a named trace for the checksum-mismatch path. |
| Target self-registration | Low — idempotent upsert with no ordering constraints. | Service-level idempotency tests; promote to a modeled flow only if create-vs-update grows additional states. |
| Storage-limit block | Medium — must never evict; the branch is small but load-bearing. | Cover with cap-enforcement service tests; fold the block branch into the backup-run flow model. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
