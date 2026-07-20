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
- behavioral, coverage, contract-compatibility, and provenance comparison axes,
  plus typed diagnostics and provider remediation;
- typed run evidence and opaque artifact IDs;
- artifact access and historical degradation diagnostics.

GCT passes the complete `PhaseDiff` sequence through without mapping phase keys
to local surfaces. Screenshots and Workflows are evidence lenses: they select
artifact kinds across every producer phase. Producer phase is provenance, not
a routing key.

## Capture and comparison

`baseline snapshot` starts one comprehensive run with the `baseline` capture
profile. Finalization persists the V2 manifest and pin idempotently, so client
disconnects and restart recovery cannot create duplicate pins. A ready baseline
means that a stable before-result was pinned and persisted; it does not mean
that every suite phase passed. A terminal failed run can therefore be valid
baseline evidence. A dirty shared workspace is normal: its run receives
`shared-scoped` provenance when the declared scenario inputs and validation
configuration were fingerprinted and stayed stable for that attempt. `strict`
provenance remains the optional clean linked-worktree tier. Only missing scope
identity or a relevant edit *during the attempt* is degraded; neither ordinary
file changes nor unrelated concurrent edits make a baseline untrustworthy. A
phase that was unavailable at capture is retained as incomplete coverage: it
makes that phase's later comparison not-comparable until it can be measured,
but never discards the before-result for the other measured phases.

`baseline diff` compares the anchored run with one resolved current run. The
result contains Test Genie's dynamic phase diffs plus typed evidence catalogs.
The comparison envelope has four independent axes: `behavior` (regression,
new failure, preexisting, cleared, clean, or unknown), `coverage` (measured or
why it was not measured), `compatibility` (including a same-key
`changed-unreviewed` validator), and `provenance` (strict, shared-scoped,
volatile, mismatched, or legacy). A required gate uses one named policy over those axes:
observed regressions/new failures fail; incomplete required coverage,
unreviewed contracts, and non-verified provenance return a distinct
not-comparable result. Provider-down, skipped, missing-artifact, and legacy
cases retain machine-readable diagnostics and remediation. Different Git SHAs
alone never make compatible, verified runs incomparable.

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
The server waits for Test Genie outside the short baseline commit critical
section. Once a run is terminal, it rechecks the manifest, pins, and saves
under that section so duplicate finalizers converge on exactly one anchor. An
in-progress capture consequently cannot delay a separate terminal capture.

## Collections and scoped source evidence

A baseline collection is a branch-scoped, durable selection of existing
single-scenario baseline identities. It records required-member coverage
(`ready`, `pending`, `failed`, `skipped`, and `stale`) rather than inventing a
multi-scenario Test Genie run. Required coverage is complete only when every
required member is ready; partial coverage is never a clean behavioral result.

```text
git-control-tower baseline collection capture --name N --member scenario[:baseline] ...
git-control-tower baseline collection show --name N --wait
git-control-tower baseline collection extend --name N --member newly-affected-scenario
git-control-tower baseline collection diff --name N --operation-id phase-1 [--member S ...]
git-control-tower baseline collection diff status --name N --operation-id phase-1
git-control-tower baseline collection diff wait --name N --operation-id phase-1
git-control-tower baseline collection delete --name N
```

Collection capture and diff are fast durable starts. Each normal CLI response
prints its exact producer-owned `show --wait` or `diff wait` command;
run that command once, and after an interruption rerun the same command rather
than polling. The server owns the durable child handles and terminal result.
Capture and diff waits reattach independent child runs concurrently with bounded
fan-out. Cancellation or deadline preserves pending handles and returns a typed
detached standing with exit `124`; terminal failure/regression uses its domain
exit code. They never return success while required work remains pending.
Ordinary collection reads are pure: they project durable parent state and child
standings without reconciling or mutating them. A collection member is
`ready` only after its immutable baseline anchor persists; it can be ready
while another member remains `pending`, and neither status asserts a passing
suite verdict.
Collection diff starts persist a caller-supplied operation identity before any
child run starts. Reusing the same identity with the same selected members
reattaches to that operation; changing the selection is rejected. `--member`
is the canonical repeatable selector (`--scenario` remains an alias), and no
selector means every collection member—the required final DoD scope.

Collection membership is append-only. Before editing a newly discovered
scenario, use `collection extend` to capture its before-state and then use the
printed collection wait command. Existing members cannot be removed, replaced,
or re-anchored. An already edited scenario cannot obtain a trustworthy before
baseline for that earlier state and must follow an explicit degraded/repair
workflow instead. This is about the missing *before behavior*, not ordinary
workspace dirtiness.

Collections may carry separately captured path snapshots. Before capture, use
the Git-aware estimate: by default it selects tracked files plus non-ignored
untracked files, rejects traversal and sensitive locations (`.git`, `.env`,
`secrets`, `credentials`), and excludes symlinks. Ignored files require
`--include-ignored`; retaining text bodies requires `--retain-content`.
Snapshots are metadata-only by default (path, mode, size, digest), while the
explicit retained-content mode remains private and bounded. New snapshots also
record source-policy version 1; zero remains the readable legacy value for
historical manifests. The estimate lists
eligible/excluded counts, projected retained bytes, contributors, issue codes,
and repair selections before any Test Genie collection member starts.

```text
git-control-tower baseline path estimate --path 'scenarios/foo/**'
git-control-tower baseline path capture --name before --path 'scenarios/foo/**' --retention 168h
git-control-tower baseline path capture --name after --path 'scenarios/foo/**'
git-control-tower baseline path diff --before before --after after --path 'scenarios/foo/**'
```

An individual `scenarios/<name>/**` selection is safe unless its measured
estimate requires repair. Whole `scenarios/**` and `packages/proto/gen/**`
selections are reported as repair-required with exact narrower paths discovered
from the estimate (not placeholder globs); source
evidence is informational and never a Test Genie verdict.

Source deltas include additions, deletions, unambiguous digest-based renames,
and metadata/content modifications, and are always labelled
`informational-source-evidence`: they are
not a Test Genie verdict and cannot make incomplete collection coverage or a
behavioral regression pass. API and CLI output contains path metadata and
digests only, never retained file bytes. The optional diff selection is
validated with the same safe repo-relative glob policy and matches either side
of a rename. Deleting a snapshot removes its
manifest and garbage-collects content objects no longer referenced by any
snapshot in that repository.

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

Collection manifests are stored under
`data/<repoID>/baseline-collections/<branch>/`; path snapshot manifests and
their content-addressed objects are stored under
`data/<repoID>/path-snapshots/`. Source object writes use private permissions,
atomic replacement, a 1 MiB per-file text cap, an 8 MiB per-snapshot retained
content cap, a 64 MiB repository source-evidence quota, and a seven-day
retention lease by default. `baseline path capture --retention` can choose a
shorter or longer whole-second lease up to 30 days; the resulting expiry is
returned by the API and CLI. Capture sweeps expired manifests under the same
store lock and garbage-collects unreferenced objects.

Writes are atomic and guarded by branch-scoped locking. Test Genie terminal
waits never hold the baseline commit mutex; concurrent finalizers for the same
capture may await the same durable run but converge on one manifest and one
pin during the short terminal commit. Collection members use the same rule, so
a terminal sibling can persist while another member is still awaiting Test
Genie. A detached HEAD is scoped using its abbreviated commit identity.
Staleness is derived from the captured git/tree identity and current worktree
state.
