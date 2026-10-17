# Diff Archive Design

## Last Updated
2026-09-27

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

1. Compute the diff with the shared generator (see §2).
2. Write the unified patch and every per-file content blob to disk through
   `BlobStore.Put`. Atomic per blob via
   `storage.WriteFileAtomic` (temp file → fsync → rename).
3. For non-committing full whole-file approval, persist the original request,
   before-file identities and retained archive in `sandbox_prepared_approvals`.
   Then apply that retained patch. This path captures both source sides in
   private staging before generating the patch and blobs.
   Reject/Delete have no source-apply step. Partial approval, hunk approval and
   turn checkpoints do not yet use this pre-apply capture.
4. Open a single SQL transaction; inside it:
   - `INSERT INTO sandbox_diff_archives ...`
   - `UPDATE sandboxes SET status = ... WHERE id = ...`
   - remove the prepared approval record, when present.
   - any other status-flip writes (e.g. `approved_at`, audit log entry).
5. `COMMIT`. The archive row and the new terminal status become
   visible together.

A full, whole-file approval MUST retain its patch and file bodies before source
writes. A capture or blob-write failure leaves source and status unchanged. A
later SQL failure rolls back the archive/status transaction, but does not undo
source application. The response reports that distinction. Prepared approval
blobs survive that failure; deleting them would erase the applied evidence.
Terminal publication is create-once: an identical archive replay succeeds;
different content, attribution, status or capture time for the same sandbox
refuses publication and preserves the original row. A nonterminal sandbox with
an existing archive refuses new application rather than replacing that evidence.
Reject/Delete clean failed capture blobs only after confirming that neither a
published archive nor a pending approval references the sandbox's blob tree.
An unavailable ownership check preserves the bytes. Retention shares the native
review lock with publication and must not delete pending approval evidence.

Source application is not part of the SQL transaction. Non-committing full
whole-file approval now has durable retry recovery. Resubmit the original request:

| Canonical source on retry | Owner action |
|---|---|
| Every affected path matches the retained after identity | Publish the original archive/status without applying again; report `applied=0` |
| Every affected path matches the retained before identity | Apply the retained patch, without reading new overlay contents |
| Mixed, divergent, missing evidence or changed request attribution | Refuse source writes; retain the original pending identity |

The before/after identities include bytes, absence, executable mode and symlink
targets. Retry cannot change attribution or approval effects. An optional expected
digest must match the retained patch. Empty and omitted selection arrays are
equivalent. Operation-derived provenance IDs permit identical replay, reject
conflicting content, and preserve later commit links.

The existing blob-store owner uses the shared platform native lock for cancellable
cross-instance approval exclusion. One lock file lives outside evictable blob
trees; it releases on process exit. Pending approval blocks Delete, Reject, Start,
Resume, Discard, Rebase and turn checkpointing. Stop can still release a mount;
its version check prevents overwriting a concurrent terminal transition. External
terminal teardown hooks run after release of the review lock.

This is not an atomic source-tree freeze. It does not stop an existing worker
from editing its overlay. Freeze/drain and preservation of later unapplied edits
must be qualified before the unattended campaign. Partial/hunk/checkpoint and
commit-producing approvals do not yet have this write-ahead recovery.

There is **no `pending` archive state**. We never commit a row that
promises content we have not yet written. The `archive_state` taxonomy
in §3 has only two values precisely so that a row's existence implies a
durable, queryable snapshot.

## 2. Snapshot reuses the live diff generator

There is one diff generator. Reject/Delete use `Service.GetDiff`. Full whole-file
approval uses the same change detector, filters, generator and sort over a private
copy of the accepted file sides. It passes that exact in-memory patch to both the
blob store and patcher. It MUST NOT regenerate the archive from a canonical lower
layer that application has already changed. The resulting `*types.DiffResult`
supplies:

- `files_json`: the per-file index (path, change_type, size, blob hash)
- `stats_json`: the aggregate stats (`filesAdded`, `filesModified`, `filesDeleted`, etc.)
- `unified_diff_path`: the gzipped blob containing the unified diff text
- per-file content blobs, including empty files, binary bytes and symlink targets

The reason is divergence containment. Two diff generators inevitably
drift: one fixes a bug, the other doesn't; one normalizes line endings,
the other doesn't; one adds a stat field, the other doesn't. With a
single generator, archives capture the generated patch without a parallel format.
Full whole-file approval additionally binds its patch and bodies to the same
captured sides. This is a sequential file capture, not an atomic source-tree
freeze. Future improvements to
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
  but the sandbox stays in its existing nonterminal state. No terminal
  transition, no snapshot.
- **Discard**: mutates the upper dir but does not transition.
- **Stop**: reversible (Stopped → Active is allowed). Not terminal.
- **Sandbox lifecycle: Creating → Error**: the sandbox never reached
  a state where any diff existed. The downstream `Error → Deleted`
  step writes the `not_captured` row.

## Reviewed-patch approval

Diff reads expose `patchSha256` (Connect: `patch_sha256`), the lowercase
SHA-256 of the exact unified-diff bytes. A captured empty patch has the
empty-content digest. Uncaptured or unavailable evidence does not receive
a fabricated digest. This identifies a patch, not the unchanged source tree.

After reviewing those bytes, pass `expectedPatchSha256` to REST approval or
`expected_patch_sha256` to Connect promotion. Both CLI approval commands accept
`--expected-patch-sha256 <digest>`. The service checks the same in-memory patch
that the patcher receives. A mismatch refuses approval before source writes;
`force` does not bypass it. Conditional approval requires `mode=all`, no file or
hunk selection, and no silent acceptance-filter exclusion. Malformed approval
JSON is an error, never a request for unconditional default approval.

The response returns `appliedPatchSha256`. Full approval persists that identity
in the existing sandbox metadata, transactionally with its terminal transition.
A matching terminal retry returns the original identity without applying again,
including after overlay deletion. A different requested digest, or an old
approval with no persisted identity, cannot receive conditional success. Existing
unconditional callers retain their behavior; delivery qualification must supply
the reviewed identity, not rely on that optional default.

This patch-only precondition is not candidate freezing or one-use qualification.
Write-ahead recovery covers non-committing full whole-file approval only; it is
not independent acceptance or a one-use qualification lease.

## Process drain on stop

Managed process-drain obligation: process admission MUST share the review-owner
lock with Stop/Start. Admission releases that lock after registration,
not after the entire command. Stop MUST drain registered processes before it
unmounts or reports stopped. Synchronous exec MUST register at start and retain
actual exit evidence. Provider maintenance MUST cover both exec entry points.
These are currently registered WSS-managed process guarantees; AM native
sessions, external writers, descendants of exited leaders and restart recovery
still require their own authoritative drain evidence. The tracker is in-memory;
an empty inventory after restart is not proof that old workers stopped.

Drain investigation: H1, driver unmount already drains managed processes; H2,
stop only runs best-effort teardown hooks and misses process termination. A real
copy-driver HTTP regression confirmed H2: a tracked sleep remained live after a
successful stop. Stop now shares the native review lock with process admission,
drains before unmount/status publication, and rechecks the drain on stopped-state
retries. Synchronous exec registers before waiting and obeys provider maintenance.
Async exit publication waits for registration; failed launch setup kills its group.
Cleanup no longer invents a SIGKILL receipt that suppresses a delayed actual exit.
Real-process regressions reproduced the live-after-stop, stopped-retry and guessed
exit defects before repair. Focused race tests and all six affected API packages
pass, including portable build checks. These tests use temporary storage; they
do not qualify AM native-session or cross-restart drain.

## Endpoint resolution

`Service.GetDiff` owns live-versus-archive selection. The REST diff endpoint and
Connect `GetSandboxDiff` both use this operation. Transports must not reimplement
the status decision or regenerate a terminal sandbox's diff from overlay paths.
Resolution is by sandbox status:

- `Active` or `Stopped` → live overlay path (today's behavior).
- `Approved`, `Rejected`, `Deleted` → archive path.

If the archive row is `complete`, the response is the same shape as the
live response, with an additional `archive_state` field set to
`complete`. If the row is `not_captured`, the response is `200 OK` with
an empty `Files` array, an empty `UnifiedDiff`, and `archive_state`
set to `not_captured`. Consumers render this as "no diff captured" —
not a 404, not an error.

A complete archive whose unified-diff blob or reader is unavailable returns an
error. A missing blob is not an empty captured diff. A genuinely captured empty
diff has a retained blob with the digest of empty content and remains valid.
Per-file bodies are still loaded on demand; the list read does not verify every
file body. A fresh review must retrieve the evidence it actually uses.

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

## Atomicity boundary

| Full whole-file approval stage | Failure behavior |
|---|---|
| Capture sides, generate patch, persist blobs | No source application or terminal publication |
| Persist write-ahead approval, then apply retained patch | Report failure; retain the original pending record; retry reconciles exact before/after state |
| Publish archive/status and consume pending approval in SQL | Roll back all three writes on failure; source may already be applied; retry the original request |

Private staging is removed when the operation returns. File bodies are processed
one at a time, not retained as an additional all-files memory buffer. This does
not bound patch size or provide crash cleanup for abandoned staging directories.

## What this design intentionally rules out

- A row whose blobs are missing (we never commit before writing).
- Full whole-file approval archiving a post-apply regeneration instead of its applied patch.
- A status flip that lands without a corresponding archive row (single transaction).
- A sandbox in History with no row (we always write `not_captured` when we cannot produce content).

## What this design accepts

- Cross-archive content duplication. Acceptable; bounded by retention.
- Pending approval records pin evidence until publication; they are not expired
  by terminal-archive retention. Blobs written before intent persistence can still
  become unindexed on interruption. Their bounded reconciliation, abandoned staging
  cleanup and bounds on unresolved intents remain open. Never erase potentially
  applied evidence merely to meet a disk limit. Content addressing deduplicates
  equal bytes, not history.
- Old blobs becoming unreadable if their archive row is evicted by
  retention while a UI request is in flight. The endpoint returns 404 in
  that race; the UI surfaces "archive expired."

## See also

- `docs/internal/STORAGE_AUDIT.md` — overall storage audit; will be
  amended by Phase 6 to record the hybrid design.
- `docs/internal/INVARIANTS.md` — system-wide invariants.
- `scenarios/prompt-manager/store/skills/packs/core/storage-steer/SKILL.md`
  §9.4 — canonical hybrid DB+filesystem pattern.
