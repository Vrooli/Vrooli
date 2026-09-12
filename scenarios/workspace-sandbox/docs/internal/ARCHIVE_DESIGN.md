# Diff Archive Design

## Last Updated
2026-04-29

## Purpose

When a sandbox transitions to a terminal state (Approved, Rejected, or
Deleted), its overlayfs is unmounted and the upper/work/merged
directories are removed. The on-demand diff generator
(`Service.GetDiff`) requires those directories, so without an archival
seam any consumer that asks for the diff after teardown receives an
empty result. Workspace-sandbox owns file-change capability for the
ecosystem; persisting the diff at terminal transition is the single
source of truth that downstream consumers (agent-manager UI, GCT) read
through.

This document records the three normative policy commitments that
govern the archive seam. Code, tests, and reviews must hold these
contracts. Drift here is the cheapest place to misalign and the most
expensive place to debug, so the rules are stated up front in plain
prose.

## 1. Snapshot before status flip, transactionally

Every snapshot runs **before** the sandbox transitions to its terminal
status. The sequence is fixed:

1. Compute the diff (via `Service.GetDiff`'s output — see §2).
2. Write every per-file content blob to disk through
   `BlobStore.Put`. Atomic per blob via
   `storage.WriteFileAtomic` (temp file → fsync → rename).
3. Open a single SQL transaction; inside it:
   - `INSERT INTO sandbox_diff_archives ...`
   - `UPDATE sandboxes SET status = ... WHERE id = ...`
   - any other status-flip writes (e.g. `approved_at`, audit log entry).
4. `COMMIT`. The archive row and the new terminal status become
   visible together.

If any step fails — blob write, repository insert, status update — the
transaction is rolled back, partial blobs are best-effort cleaned via
`BlobStore.DeleteSandbox`, and the sandbox stays in its pre-terminal
status (Active, Stopped, etc.). The operator retries; nothing observed
the half-committed state.

There is **no `pending` archive state**. We never commit a row that
promises content we have not yet written. The `archive_state` taxonomy
in §3 has only two values precisely so that a row's existence implies a
durable, queryable snapshot.

## 2. Snapshot reuses `Service.GetDiff` output verbatim

The snapshot path **does not** generate diffs independently. It calls
the same internal diff path that serves live `GET /diff` requests — same
status checks, same change detector, same generator,
same filters, same sort. The returned `*types.DiffResult` is serialized
byte-for-byte into:

- `files_json`: the per-file index (path, change_type, size, blob hash)
- `stats_json`: the aggregate stats (`filesAdded`, `filesModified`, `filesDeleted`, etc.)
- `unified_diff_path`: the gzipped blob containing the unified diff text
- per-file content blobs, one per non-empty `FileChange`

The reason is divergence containment. Two diff generators inevitably
drift: one fixes a bug, the other doesn't; one normalizes line endings,
the other doesn't; one adds a stat field, the other doesn't. With a
single generator, archives capture exactly what the live endpoint would
have served at the moment of transition. Future improvements to
`GetDiff` automatically apply to the live path; archives stay stable
byte-for-byte because they are immutable artifacts on disk.

The corollary is that any test which exercises the live diff path also
exercises the archived shape. There is no parallel "archive-only"
golden file format to maintain.

## 3. `archive_state` taxonomy

`archive_state` is a TEXT column on `sandbox_diff_archives` with
exactly two valid values:

- **`complete`**: the snapshot ran, blobs are on disk, and the
  metadata row is consistent with them. The endpoint serves the diff
  by reading the blobs through `BlobStore.Get`.
- **`not_captured`**: the snapshot was deliberately skipped (see
  below). The metadata row exists so the History UI can render an
  explicit "no diff captured" state for the sandbox; no blobs exist
  on disk, `unified_diff_path` is `NULL`, and `total_blob_bytes` is
  `0`.

We commit **no other states**. There is no `pending`, no `failed`, no
`partial`. Snapshot failure aborts the transition (see §1), so a
terminal-status sandbox with no archive row is impossible by
construction.

### When `not_captured` applies

- **`Error → Deleted`**: the overlay is typically unsalvageable
  (process crashed, mount lost, upper dir corrupted). We still write
  an archive row so the sandbox appears in History, but the row
  carries `archive_state="not_captured"`.
- **`CanGenerateDiff(sandbox) == false`** at snapshot time: the
  sandbox cannot produce a diff (no upper dir, lower dir vanished).
  Same handling: row exists, `not_captured`, no blobs.

### When snapshots are skipped entirely (no row)

- **Partial Approve**: the call returns a partial-acceptance result
  but the sandbox stays Active or in NeedsReview. No terminal
  transition, no snapshot.
- **Discard**: mutates the upper dir but does not transition.
- **Stop**: reversible (Stopped → Active is allowed). Not terminal.
- **Sandbox lifecycle: Creating → Error**: the sandbox never reached
  a state where any diff existed. The downstream `Error → Deleted`
  step writes the `not_captured` row.

## Endpoint resolution

`GET /api/v1/sandboxes/{id}/diff` is the single front door for both
live and archived diffs. Resolution is by sandbox status:

- `Active` or `Stopped` → live overlay path (today's behavior).
- `Approved`, `Rejected`, `Deleted` → archive path.

If the archive row is `complete`, the response is the same shape as the
live response, with an additional `archive_state` field set to
`complete`. If the row is `not_captured`, the response is `200 OK` with
an empty `Files` array, an empty `UnifiedDiff`, and `archive_state`
set to `not_captured`. Consumers render this as "no diff captured" —
not a 404, not an error.

For live responses (`Active`, `Stopped`, `Creating`, `Error`), the
`archive_state` field is **omitted** (zero value). Consumers
distinguish three explicit states by the field:

- **field absent / empty** → live overlay; trust `Files`/`UnifiedDiff`
  as real-time data (or empty for the no-overlay edge cases).
- **`"complete"`** → archive snapshot; `Files`/`UnifiedDiff` reflect
  what was captured at terminal transition.
- **`"not_captured"`** → archive row exists but no content was
  captured; render "no diff captured".

This three-way taxonomy lets the CLI and UI label the source
unambiguously without falling back to inference (e.g. checking sandbox
status separately).

## Storage layout

Per `storage-steer §9.4` (hybrid DB + filesystem):

- **Metadata** lives in SQLite (`sandbox_diff_archives` table). It is
  small, queryable, transactional with the sandbox status flip, and
  listable for retention.
- **Content** lives on disk under `storage.ClassData`, scoped by
  scenario, content-addressed by SHA-256:

  ```
  <ClassData>/<app>/workspace-sandbox/archives/<sandbox_id>/<sha256>.gz
  ```

  Content is gzipped. The `unified_diff_path` blob and the per-file
  blobs live in the same per-sandbox directory so retention can drop
  the directory in one operation when an archive is evicted.

Per-sandbox content addressing means identical files inside one sandbox
dedupe naturally (e.g. an empty file across many directories shares one
blob). We do **not** dedupe across sandboxes in v1; cross-archive
dedup risks cascading invalidation when retention deletes a sandbox
that held the only copy of a blob another archive references. v1 is
intentionally simple: per-sandbox isolation, drop-in retention.

## Atomicity boundary diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ snapshotDiff(ctx, tx, sandbox)                                  │
│                                                                 │
│ 1. result := computeDiff(ctx, sandbox)        // §2: reuse      │
│                                                                 │
│ 2. for each file in result.Files:                               │
│       hash := blobstore.Put(sandboxID, content)                 │
│       index[path] = hash                       // disk-durable  │
│                                                                 │
│ 3. unifiedHash := blobstore.Put(sandboxID, unifiedDiff)         │
│                                                                 │
│ 4. archiveRepo.Insert(tx, archive)             // SQL row       │
│                                                                 │
│ Caller continues inside the same tx:                            │
│ 5. repo.UpdateStatus(tx, sandbox, terminalStatus)               │
│ 6. tx.Commit()                                                  │
│                                                                 │
│ On failure at any step:                                         │
│   - tx.Rollback() (steps 4–6)                                   │
│   - blobstore.DeleteSandbox(sandboxID) (best-effort cleanup)    │
│   - status stays pre-terminal                                   │
└─────────────────────────────────────────────────────────────────┘
```

## What this design intentionally rules out

- A row whose blobs are missing (we never commit before writing).
- A blob whose row is missing for a `complete` archive (cleanup-on-rollback removes orphans).
- A live diff that disagrees with its archive at the moment of transition (single generator).
- A status flip that lands without a corresponding archive row (single transaction).
- A sandbox in History with no row (we always write `not_captured` when we cannot produce content).

## What this design accepts

- Cross-archive content duplication. Acceptable; bounded by retention.
- A best-effort cleanup that fails to remove orphan blobs after rollback.
  Next snapshot for the same sandbox ID overwrites them by hash; retention
  sweeps them eventually. Not a correctness issue.
- Old blobs becoming unreadable if their archive row is evicted by
  retention while a UI request is in flight. The endpoint returns 404 in
  that race; the UI surfaces "archive expired."

## See also

- `docs/internal/STORAGE_AUDIT.md` — overall storage audit; will be
  amended by Phase 6 to record the hybrid design.
- `docs/internal/INVARIANTS.md` — system-wide invariants.
- `scenarios/prompt-manager/store/skills/packs/core/storage-steer/SKILL.md`
  §9.4 — canonical hybrid DB+filesystem pattern.
