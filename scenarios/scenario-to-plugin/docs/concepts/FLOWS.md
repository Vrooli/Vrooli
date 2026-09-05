# Flows — Scenario to Plugin

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

`health` is a stateless reporting domain and ships no workflows. Every
other domain in this scenario is a stage in one long-running pipeline, so
all of them carry ordered state.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Package build | composition | Operator or CI requests a package for a scenario at a source commit. | A materialized Agent Plugins tree, or a typed composition failure. | Ordered build with partial-artifact cleanup; not retryable in place — a retry produces a new package. | Level 3 (matrix + traces) at first release; Level 4 contract once the shape is stable. |
| Conformance run | conformance | A package reaches `composed`. | A pass/fail record with per-rule findings. | Ordered rule execution, fail-closed, no partial pass. | Level 3. |
| Attestation run | attestation | A package reaches `conformant`. | Scanner verdict, signature, provenance, and SBOM bound to a digest. | Ordered and non-resumable; a failed attestation restarts from scan. | Level 4 — external side effects make illegal transitions expensive. |
| Rehearsal | rehearsal | A package reaches `attested`. | A protocol-profile journey manifest with gate dispositions. | Long-running, cancellable, with sandbox teardown guaranteed on every exit path. | Level 5 — the only flow whose failure can leave an external resource allocated. |
| Publication | distribution | Operator publishes a rehearsed package. | Per-channel confirmed publication, or a refusal naming the closed gate. | Multi-channel fan-out with partial success, confirmation-by-retrieval, and no implicit rollback. | Level 5 — outward-facing and not fully reversible. |
| Revocation | distribution | Operator revokes a published version. | Per-channel withdrawal outcomes, possibly partial. | Fan-out over recorded history; idempotent; partial failure is a terminal reportable state. | Level 5. |

## Flow Details

Every pipeline flow shares one contract: **a stage may only start from
the terminal success state of the stage before it, and no stage may
report success without a durable record.** The two rules together are
what make the ladder in the UI a true statement rather than a rendering.

### Package build

- Owner domain: composition.
- Trigger: `package build <scenario>` from CLI, UI, or CI.
- Inputs: scenario slug, source commit, validated declaration snapshot.
- Steps:
  1. Resolve and snapshot the declaration (via `declaration`).
  2. Refuse if readiness reports a blocking prerequisite.
  3. Materialize `plugin.json` at the plugin root.
  4. Materialize each declared skill under `skills/<name>/`.
  5. Materialize `mcp.json` when an MCP server is declared.
  6. Digest the tree and write it to the capture store.
  7. Record the package and its component inventory.
- Outputs: a `packages` row in state `composed`, plus a digest-addressed
  artifact tree.
- Failure modes: absent or invalid declaration, blocking prerequisite,
  unrepresentable component, capture-store write failure.
- Retry/cancel: a failed build is terminal. Retrying produces a new
  package rather than resuming, so a partially written tree can never be
  mistaken for a complete one. Partial artifacts are removed on failure.

### Conformance run

- Owner domain: conformance.
- Trigger: a package in state `composed`.
- Inputs: package digest, artifact tree, pinned `cli-manifest` revision.
- Steps:
  1. Pin the wrapped scenario's `cli-manifest` revision and record it.
  2. Run skill-specification rules (frontmatter, name/folder, limits).
  3. Run injection rules (hidden Unicode, bidi marks, NFC, angle brackets).
  4. Run permission rules over `allowed-tools`.
  5. Run install-script rules (pinning, checksum, elevation, prefix).
  6. Resolve every documented command against the pinned manifest.
  7. Record the run and every finding with file, offset, and rule id.
- Outputs: package advances to `conformant`, or to `failed_conformance`
  with a complete finding list.
- Failure modes: any rule failure. There is no warning tier — a rule that
  cannot decide is a failure, not a pass.
- Retry/cancel: all rules run even after the first failure, so one pass
  yields the full finding list. Re-running requires a new package.

### Attestation run

- Owner domain: attestation.
- Trigger: a package in state `conformant`.
- Steps:
  1. Refuse if the conformance record is absent or failing.
  2. Run configured scanners; a critical or high finding fails the run.
  3. Check for credential literals in the tree, SBOM inputs, and metadata.
  4. Sign the digest through the managed release authority.
  5. Attach SLSA provenance naming source commit and build environment.
  6. Generate and attach a CycloneDX SBOM.
- Outputs: package advances to `attested` with signature, provenance, and
  SBOM references.
- Failure modes: ordering violation, scanner finding, credential literal,
  signing-authority unavailability.
- Retry/cancel: non-resumable. A failed attestation restarts from the
  scan step so that no signature is ever produced over a partially
  re-verified tree.

### Rehearsal

- Owner domain: rehearsal.
- Trigger: a package in state `attested`.
- Steps:
  1. Create a `workspace-sandbox` instance with no Vrooli runtime access.
  2. Run the install script; record acquisitions.
  3. Run the install script a second time; require a no-op.
  4. Compare acquisitions against the declaration; fail on any undisclosed one.
  5. Execute each documented command; capture redacted output and exit status.
  6. Prove the entitlement sign-in path when the skill declares one.
  7. Emit a protocol-profile journey manifest with gate dispositions.
  8. Tear down the sandbox.
- Outputs: package advances to `rehearsed` with a journey manifest, or to
  `failed_rehearsal` with the failing gate named.
- Failure modes: sandbox creation failure, install failure, non-idempotent
  install, undisclosed acquisition, command failure, timeout.
- Retry/cancel: cancellable at any step. **Teardown runs on every exit
  path including cancellation and panic** — a leaked sandbox is the one
  failure here that costs real resources. A cancelled rehearsal is
  `unavailable`, never `failed`: the package was not judged.

### Publication

- Owner domain: distribution.
- Trigger: operator publishes a package in state `rehearsed`.
- Steps:
  1. Report the `TargetVerdict` to `deployment-manager` (references only).
  2. Request the release decision for this exact source commit and target.
  3. Refuse if no passing decision is returned.
  4. For each selected channel, resolve credential references and push.
  5. Confirm by retrieving the artifact at the published digest.
  6. Record a per-channel outcome, including failures.
- Outputs: per-channel `published` or `failed`; package state `published`
  when at least one channel confirmed.
- Failure modes: missing or non-matching release decision, credential
  reference unresolvable, push failure, confirmation failure.
- Retry/cancel: retry is per-channel and idempotent by digest. There is
  **no implicit rollback** — a channel that succeeded stays published, and
  the operator decides whether to revoke.

### Revocation

- Owner domain: distribution.
- Trigger: operator revokes a published version.
- Steps:
  1. Derive the channel set from recorded publication history.
  2. Attempt withdrawal in each channel.
  3. Record each channel's outcome.
  4. Mark the revocation complete only when every channel confirmed.
- Outputs: `revoked`, or `revoked_partial` naming each channel that still
  carries the artifact.
- Failure modes: a channel that does not support withdrawal; a channel
  that is unreachable.
- Retry/cancel: idempotent and re-runnable. `revoked_partial` is a real
  terminal state, not a transient one — some registries cannot hard-delete
  a version, and reporting that truthfully is what lets an operator
  escalate to that registry's process.

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| composition / package build | `pending`, `composing`, `composed`, `failed_composition` | Any escape from a terminal state; `composed` without a recorded artifact digest | `flow.json` contract, generated model, replay tests |
| conformance / run | `pending`, `checking`, `conformant`, `failed_conformance` | `conformant` from any state other than `checking`; `conformant` with an unresolved finding | `flow.json` contract, generated model, replay tests |
| attestation / run | `pending`, `scanning`, `signing`, `attested`, `failed_attestation` | `signing` before a passing scan; `signing` when the conformance record is absent or failing; resume into `signing` after a failure | `flow.json` contract, generated model, ordering tests |
| rehearsal / run | `pending`, `provisioning`, `installing`, `exercising`, `rehearsed`, `failed_rehearsal`, `cancelled`, `unavailable` | Any terminal state reached without sandbox teardown; `rehearsed` with a failing required gate; `failed_rehearsal` for a cancellation | `flow.json` contract, generated Quint model, teardown side-effect tests, replay |
| distribution / publication | `pending`, `verdict_reported`, `gate_requested`, `publishing`, `published`, `published_partial`, `refused` | `publishing` without a passing decision for the same commit; `published` before retrieval confirmation | `flow.json` contract, generated Quint model, gate-refusal replay |
| distribution / revocation | `pending`, `withdrawing`, `revoked`, `revoked_partial` | `revoked` while any channel outcome is unconfirmed; deleting a publication row as part of revocation | `flow.json` contract, generated Quint model, fan-out replay |

Two invariants hold across every machine above and are checked in
`CheckInvariants` rather than per-transition:

- **Monotonic ladder.** A package's stage index never decreases. There is
  no path that returns a `published` package to `composed`.
- **No green from absence.** A gate disposition of `passed` requires a
  recorded evidence reference. A missing check is `unverified`, which is
  never treated as a pass.

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
| Curated registry submission and review tracking | Low. External review is human-paced and currently manual; modeling it before running it once would encode a guessed process. | Model after one package is accepted by hand (`OT-P2-001`). |
| Composite package build across several scenarios | Medium. Multiplies both the drift surface and the revocation fan-out. | Model only after a single-scenario package has been published and measured (`OT-P2-002`). |
| Automated re-verification of published versions on CLI change | Medium. A published skill can drift after publication when the wrapped CLI moves; nothing currently re-checks it. | Depends on `manifest_pins` being populated in production. Record as a real gap in `PROBLEMS.md` rather than an assumed feature. |
| Scheduled artifact and rehearsal-log pruning | Low. Retention rules are documented in `DATA.md` but not enforced by a job. | Implement with the first production deployment. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
