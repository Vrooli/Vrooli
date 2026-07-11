# Temporal Flows & Async Patterns

## Last Updated
2026-07-10

## Async Flows Identified

| Flow | Entry Point | Async Operations | Completion Signal |
|------|-------------|------------------|-------------------|
| Baseline snapshot | `baseline snapshot` / `SnapshotForBaseline` | One comprehensive Test Genie run, terminal snapshot hydration, exactly-once pin, V2 manifest commit | Ready capture intent plus V2 manifest with `migration.pin_reconciled_at` |
| Baseline diff | `baseline diff` / `StartDiff` | Current Test Genie run and descriptor-aware comparison | Ready diff intent plus durable comparison cache |
| V1 migration | Baseline read/list | Validate one-run identity, rewrite V2 atomically, reconcile owner-scoped pin | V2 manifest whose reconciliation checkpoint is durable |

## Race Conditions

- Concurrent migration/list/delete serializes through baseline storage locks.
  Pin reconciliation is owner-idempotent; only the successful pin is followed
  by `pin_reconciled_at`.
- A failed unpin during deletion leaves the manifest present so retry cannot
  orphan an untracked pin.
- Snapshot or diff caller cancellation cannot publish a false terminal result;
  detached work owns its own context and commits only after terminal Test Genie
  truth is available.

## Timing Assumptions

- Snapshot/diff detached tails have a 30m attachment ceiling. Expiry leaves the
  durable intent pending; it is not an execution verdict.
- The baseline CLI transport ceiling is 30m and the API write timeout is 31m,
  preventing the server from cutting off the client first.
- Test Genie owns queue and execution truth. A later status/resume attachment
  gets a fresh transport budget rather than subtracting prior queue residence.

## Checkpoint Flows

- **Snapshot:** `StartCapture` persists a pending intent before the handler
  dispatches its detached finalizer. Terminal Test Genie truth → one pin + V2
  manifest + ready intent. Resume through `snapshot status --run R [--wait]` or
  startup reattachment. Caller cancellation/deadline leaves the intent pending.
- **Diff:** `StartDiff` persists the base/current run identities before detached
  comparison. Terminal comparison cache + ready intent is the commit boundary.
  Resume through `diff status --run R [--wait]`; an absent cache is recomputed
  from durable Test Genie runs. Caller cancellation/deadline stays non-terminal.
- **CLI wait:** one blocking read. EOF, cancellation, or transport deadline
  permits exactly one inspect read by the same durable run ID; ordinary errors
  are returned and mutations are never replayed.

## Concurrency Concerns

Baseline manifest replacement is atomic. In-process locks prevent duplicate
capture/migration/delete mutations for one identity, while Test Genie's
owner-scoped pins provide the cross-retry idempotency key. Multi-scenario waits
use bounded concurrency and one `diff wait-all` attachment rather than polling.
