# Baseline Model

Git Control Tower baselines are immutable anchors to one comprehensive,
server-owned Test Genie run. They answer whether a current result regressed
from a known point without copying artifacts or inventing a second phase model.

## Schema V2

A manifest records baseline identity, scenario and branch metadata, captured
git and tree identity, and one `RunAnchor`. The anchor contains the run ID,
capture profile, phase-set digest, and the persisted Test Genie descriptor
snapshot identity. GCT pins that run once and unpins it once when the baseline
is deleted.

Test Genie remains authoritative for:

- the dynamic phase catalog and captured phase descriptors;
- comparison verdicts and typed comparison reasons;
- typed run evidence and opaque artifact IDs;
- artifact access and historical degradation diagnostics.

GCT passes the complete `PhaseDiff` sequence through without mapping phase keys
to local surfaces. Screenshots and Workflows are evidence lenses: they select
artifact kinds across every producer phase. Producer phase is provenance, not
a routing key.

## Capture and comparison

`baseline snapshot` starts one comprehensive run with the `baseline` capture
profile. Finalization persists the V2 manifest and pin idempotently, so client
disconnects and restart recovery cannot create duplicate pins.

`baseline diff` compares the anchored run with one resolved current run. The
result contains Test Genie's dynamic phase diffs plus typed evidence catalogs.
Visual deltas are advisory. A missing, legacy, corrupt, or otherwise degraded
descriptor/evidence record produces an explicit not-comparable result rather
than a guessed verdict.

The CLI commands are:

```text
git-control-tower baseline snapshot --scenario S --name N
git-control-tower baseline snapshot status --scenario S --name N --run R
git-control-tower baseline diff --scenario S --name N
git-control-tower baseline diff status --scenario S --name N --run R
git-control-tower baseline list --scenario S
git-control-tower baseline show --scenario S --name N
git-control-tower baseline delete --scenario S --name N
```

Snapshot and diff operations are durable. Canceling a client detaches; it does
not abort the Test Genie run. The status commands reattach to persisted intent.

## V1 migration

The storage boundary alone understands the former surface-pointer schema:

| Stored state | Outcome |
|---|---|
| V2 single-run manifest | Read directly; migration is a no-op. |
| Complete V1 manifest whose five pointers reference one run | Atomically rewrite as one V2 run anchor, reconcile the idempotent Test Genie pin, then persist `pin_reconciled_at`. |
| V1 pointers containing different run IDs | Reject as mixed-run and require recapture. |
| Empty, partial, or skipped V1 pointers | Reject as incomplete and require recapture. |
| Interrupted save/pin sequence | Reconcile persisted intent idempotently on recovery. |
| Historical run missing descriptor or evidence metadata | Preserve identity and report explicit degraded reasons. |

V1 is not exposed as a second API and cannot be edited into a V2 baseline.
The pin checkpoint is written only after Test Genie accepts the retention
owner. A crash before that checkpoint safely retries the same owner; a pin
failure leaves the migrated manifest explicitly unreconciled and retryable.
Deletion follows the inverse ordering: Test Genie must accept the idempotent
unpin before the manifest is removed, so a transport failure retains the
baseline identity needed for a safe retry instead of leaking an orphan pin.

Before live migration, rehearse the real manifest population through an
isolated copy with:

```text
GCT_BASELINE_REHEARSAL_SOURCE=<data>/<repo-id>/baselines \
  go test ./internal/baseline -run TestCopiedBaselineMigrationRehearsal -v
```

The rehearsal never opens the source through `Storage`; it copies each direct
scenario/branch manifest into test-owned temporary storage, runs the real
migration and retention-reconciliation path twice, and reports migratable,
already-V2, mixed, incomplete, corrupt, and simulated-pin counts.

## Storage and concurrency

Manifests remain branch-scoped under:

```text
data/<repoID>/baselines/<scenario>/<branch>/<name>.json
```

Writes are atomic and guarded by branch-scoped locking. Concurrent finalizers
for the same capture converge on one manifest and one pin. A detached HEAD is
scoped using its abbreviated commit identity. Staleness is derived from the
captured git/tree identity and current worktree state.
