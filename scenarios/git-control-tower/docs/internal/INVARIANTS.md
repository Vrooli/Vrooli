# System Invariants

## Last Updated
2026-07-10

## Critical Invariants

| Invariant | Domain Concept | Enforcement | Test Coverage |
|-----------|----------------|-------------|---------------|
| A baseline names exactly one immutable Test Genie run. | Baseline V2 identity | `BaselineManifest.Validate`, storage migration, removed create/edit/selective APIs | `internal/baseline/service_test.go`, handler and CLI tests |
| GCT never routes on Test Genie phase keys. | Descriptor-driven evidence | Lossless `PhaseDiff`/`RunInfo` forwarding and production-code phase-registry guard | `handlers/evidence/phase_agnostic_guard_test.go` |
| Artifact access is path-free and run-scoped. | Typed evidence | Test Genie opaque IDs plus GCT same-origin proxy; no provider path in proto/UI contracts | Evidence handler and artifact renderer tests |
| UNKNOWN, degraded, missing, or not-comparable evidence cannot become PASS. | Honest comparison | Baseline result aggregation and durable operation status/exit codes | Baseline service/handler/CLI tests |
| A started collection diff remains interpretable after collection cleanup. | Durable parent ownership | The operation stores its creation-time collection snapshot; status/reconciliation falls back to it, and collection deletion does not erase operation records. | `internal/baseline/collection_service_test.go` snapshot-after-delete regression |

## Important Invariants

- The review UI exposes exactly Overview, Baselines, Metrics, Screenshots,
  Workflows, Tests, AI Changes, and Agent. Persisted legacy tab values migrate
  to a retained destination; Rules and Code Quality are not routable tabs.
- Screenshots and Workflows select stable artifact kinds across every producer
  phase. Producer phase is preserved as provenance only.
- Visual differences remain advisory unless an owning provider emits a separate
  failing finding.

## Replay/Idempotency Invariants
[Operations that must be safe to retry]

- **One in-progress test-genie run per scenario.** At most one in-progress run
  exists per scenario at a time. Different scenarios run concurrently; the same
  scenario does not. This is a **correctness** requirement, not just efficiency:
  every run brings the target up via `targetruntime.EnsureRunning`, which shares
  one live instance (ports, DB, fixtures) with no mutual exclusion, and one run's
  cleanup can tear the scenario down under another. Enforced centrally in
  test-genie `runManager.Start` (atomic scan+decide+insert under one lock).
- **Run admission key** (coalescing identity): `(scenario, preset, captureProfile,
  sorted phases, sorted skip, scenarioPath, logicalRepoRoot, logicalScenarioRelPath)`.
  The baseline *name* is deliberately NOT part of it, so many diffs of one
  scenario (different `--name`) coalesce onto one comprehensive run and each
  caches its own comparison.
- **Coalesce vs reject.** An identical in-flight request (same key) **coalesces**
  onto the running run (`coalesced=true`) — idempotent, no second suite. A
  divergent request (different key) while a run is in flight is **rejected** with
  `FailedPrecondition` carrying the in-flight run id + preset, never coalesced and
  never stacked. Snapshots and diffs share the comprehensive+baseline key, so they
  coalesce with each other; only a genuinely different shape (e.g. `execute
  --preset quick`) rejects.
- **retireGrace exclusion.** A just-finished (terminal) run lingers in the
  registry for `retireGrace` (60s) so late status/follow calls read the live
  snapshot. Admission ignores these terminal lingerers — they never block or
  coalesce a fresh start.
- **Clean-tree run reuse.** A diff reuses a completed `comprehensive`+`baseline`
  run only when the working tree is clean, its sha matches exactly, and it
  finished within `GCT_DIFF_RUN_REUSE_TTL` (default 15m). A dirty tree always
  runs fresh (uncommitted edits aren't captured by sha).
- **DiffResult cache commit boundary.** The cached `DiffResult` (keyed
  `(repoID, scenario, branch, name, runID)`) is written only on a successful
  `FinalizeDiff` (atomic tmp+rename). If the finalize tail is lost (crash),
  `GetDiffResult` recomputes once on demand from the durable runs — the runs
  themselves never depend on the cache.
- **Attachment failure is non-terminal.** `context.Canceled` and
  `context.DeadlineExceeded` from a snapshot/diff wait leave the intent pending.
  They never publish a failed snapshot or ready/not-comparable diff. The CLI may
  perform one non-blocking recovery read by the same run ID after unexpected EOF;
  it never retries `StartCapture`/`StartDiff` from that path.
- **Collection-diff identity is immutable.** A collection diff operation
  snapshots the selected collection's baseline membership at creation. Its
  status/reconciliation path uses that snapshot if the mutable collection was
  later deleted; old operations without a snapshot fail explicitly and require
  recapture rather than comparing against a replacement collection by name.
- **Operation standing is the only lifecycle decision surface.** Test Genie
  owns child execution standing; GCT owns the persisted aggregate standing and
  projects child standings unchanged. `preparing` with no active phase is
  waitable, and a terminal child awaiting fan-in is `finalizing`, never a
  generic pending/blocked state.
- **Status never waits; collection-diff status reconciles terminal evidence.**
  `GetCollectionDiffStatus` performs only non-blocking reads of Test Genie's
  durable run record and projects a terminal child through its persisted diff
  intent. It never waits for active execution. A missing child intent becomes a
  terminal infrastructure failure rather than a permanent pending member.
  `WaitCollectionCapture` and `WaitCollectionDiff` remain the only blocking
  attachment operations. A timeout detaches with `directive=wait` and leaves
  durable execution untouched.
- **Reconciliation has a server-owned liveness backstop.** Every 30 seconds
  the GCT server enumerates nonterminal collection-diff operations for every
  registered repository and performs the same non-blocking durable-child
  projection. Completion therefore does not depend on event delivery or a
  detached client returning to issue status.
- **Pre-run dispatch is leased and bounded.** The complete selected child graph
  persists before Test Genie is contacted. A pending child without a run ID is
  in dispatch, carries an attempt count and lease, and is retried by status
  reconciliation after the lease expires. Three failed dispatch attempts become
  a terminal infrastructure failure with the recorded cause; they never remain
  indistinguishable from queued Test Genie work.
- **A run-backed child is never parent-queued.** Once a child run ID is
  committed, its durable lifecycle is `awaiting_child` (then `reconciling` and
  a terminal projection). Test Genie may report its own admission queue, but
  GCT exposes that attached child as executing; an operation cannot regress to
  a fresh parent-side `queued` handoff.
- **Baseline wait is outside the commit mutex.** `FinalizeCapture` can await
  any durable Test Genie run concurrently. Only the terminal recheck, pin, and
  manifest/intent write hold `captureMu`, so one stalled capture cannot block
  an unrelated terminal capture. Duplicate finalizers still converge on one
  owner-scoped pin and one manifest.
- **Legacy retention reconciliation is checkpoint-after-effect.** A migratable
  V1 manifest is atomically rewritten to a V2 run anchor, then the service asks
  Test Genie for the idempotent `gct:baseline:<name>` pin, and only after success
  persists `migration.pin_reconciled_at`. A failure or crash before the
  checkpoint retries the same owner. Delete uses the inverse safe ordering:
  successful idempotent unpin precedes manifest removal, so a failed transport
  cannot orphan a pin without the recovery identity needed to retry.

## Enforcement Mechanisms
[How each invariant is protected]

- One-run-per-scenario + coalesce/reject/retireGrace: test-genie
  `internal/runmanager/manager.go` (`Start`, `admissionKey`), tested in
  `manager_test.go` (concurrent-identical, divergent-reject, retireGrace,
  TOCTOU race under `-race`).
- Run reuse + diff cache + recompute: `internal/baseline/service.go`
  (`resolveCurrentRun`, `FinalizeDiff`, `GetDiffResult`) + `storage.go`
  (`SaveDiffResult`/`LoadDiffResult`), tested in `service_test.go`.
- Attachment detachment + one-read EOF recovery: `service.go::FinalizeCapture`/
  `FinalizeDiff` and `cli/domains/baseline/register.go::durableReadWithEOFRecovery`,
  tested under `-race` in their adjacent test files.
- Legacy migration pin/unpin ordering: `internal/baseline/service.go`
  (`reconcileMigrationPin`, `Delete`) plus the persisted `MigrationInfo`
  checkpoint, tested by the legacy migration, failure/retry, concurrent-read,
  copied-real-data rehearsal, and delete-retry tests in `service_test.go`.
