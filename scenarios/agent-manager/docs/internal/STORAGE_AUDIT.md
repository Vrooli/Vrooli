# Agent Manager Storage Architecture Audit

## Last Updated

2026-09-14

## Current Pattern

- [ ] Per-domain schema files (canonical)
- [x] Centralized schema

## Migration Strategy

- [x] Greenfield / personal local state — declarative schema plus a one-shot
  external migration script when local data must be preserved
- [ ] Brownfield versioned migrations

The application never mutates stored data at startup. On 2026-07-11, the local
development database was backed up and converted from legacy profile columns to
the role-only profile schema with a transactional script under
`/tmp/agent-manager/`. The script is intentionally not tracked.

## Architecture Status

- [ ] All domains own their SQL schema
- [x] Repository interfaces are present for primary data access
- [ ] System home contains only cross-cutting storage

## Issues Found

1. `api/internal/database/schema.sql` centrally defines domain tables; the
   storage-manager validator recommends splitting these into domain-owned schema
   providers.
2. The 2026-09-04 `storage-manager validate prove-isolation agent-manager`
   gate is still false and names `database.Open`, `database.EnsureSchemas`, and
   `RoutedRoots.Pick` as missing routed test-isolation seams.
3. The full storage validator reports 59 findings: approximately 255 MB outside
   declared roots, direct filesystem writers, one unproven creation mode, and
   two direct-SQL handler sites. These findings predate conversation search and
   remain authoritative debt rather than exceptions for new code.
4. `storage-manager declare inspect agent-manager` reports 5,589,508,864
   observed bytes covered by the current 6 GiB storage budgets.

## Conversation Search Decision

The `conversationsearch` domain will own only new regenerable projection tables
and indexes in a co-located `schema.sql`; it will not add columns to canonical
run/event tables. SQLite FTS remains the portable lexical floor. Qdrant is an
optional semantic companion whose collection name is resolved with
`storage.Collection("conversation-search")` so live and shadow variants remain
isolated. Semantic indexing is restricted to prose and quoted prose; explicit
tool-event recall remains a bounded SQLite text/regex operation and does not
amplify the vector corpus. Failed shadows are rolled back. Successful retired
vector generations are deliberately retained for reviewed rollback rather
than deleted automatically. Destructive search playbooks are not authorized
until the scenario's routed isolation gate passes.

The request/outcome telemetry table is content-free, automatically reclaims
rows older than 30 days, and caps retained rows at 100,000. Reclaim runs every
256 successful telemetry appends so it does not add a full retention query to
every search.

## Measured Conversation-Search Growth — 2026-09-04

The pre-feature live database was 1,241,026,560 bytes. During the first live
shadow exercise, indexing every tool call/result expanded the candidate to
about 554,974 SQLite documents; the candidate table occupied about 297 MB at
194,655 staged rows and repeated rolled-back writes grew the SQLite file to
about 2.37 GB. Qdrant had reached 5,249 points before the candidate was rolled
back. This was rejected as a vector-corpus policy, not accepted as a budget.

The correction keeps tool events available only in the SQLite projection and
limits semantic vectors to conversational prose/quoted prose. The live retry
must record final catalog/FTS/vector counts and physical bytes in
`docs/internal/PERFORMANCE.md`. Until that evidence exists, the existing 6 GiB
owned-data ceiling is an alarm ceiling, not proof that the projection is
efficient. Raw SQL deletion, raw Qdrant collection deletion, and an ad-hoc
whole-file `VACUUM` remain prohibited recovery advice; the only sanctioned
whole-file rewrite is the fenced owner compaction below.

## Storage Self-Management — 2026-09-14

On 2026-09-14 the live file was 293 GB with 285 GB on the SQLite freelist and
8.7 GB of live btrees. The file was created with `auto_vacuum=NONE`, so every
page freed by retention, generation pruning, or rollback stayed in the file.
The api-core DSN sets `journal_mode(WAL)` on open, which writes the header, so
`PRAGMA auto_vacuum` can never take effect on a new file without a `VACUUM`.

`internal/storagehealth` owns the file's footprint. It never deletes rows.

- **Measurement.** `GET /api/v1/storage/health` and `agent-manager storage
  status` read `page_count`, `freelist_count`, `auto_vacuum`, and the file and
  WAL sizes against the declared `storage.entries.data.budget.max_bytes`. The
  budget is read through api-core `LoadOwnerInventory` (340 ms, cached for an
  hour). The level is `ok` below 70% of the budget, `reclaim` from 70%, and
  `alarm` from 85%. The action is `none`, `incremental_vacuum`,
  `compaction_required`, or `retention_or_budget`.
- **Continuous reclamation.** Reconciler step 9 calls `Maintain` each cycle.
  In incremental mode a drain starts when free pages exceed max(64 MiB, 5% of
  live bytes), or any amount above the floor at the reclaim level, and runs
  down to a 16 MiB floor. It uses adaptive `incremental_vacuum` batches that
  each hold the writer for about 250 ms, for at most 5 s per cycle. A fixed
  2048-page batch held the writer for about 4 s on the rehearsal copy. When
  only a compaction, retention, or a budget decision can help, `Maintain`
  logs a throttled warning.
- **WAL bound.** Every pool connection sets `journal_size_limit` to 64 MiB, so
  the WAL no longer keeps its high-water size (1 GB was observed on
  2026-09-14).
- **Reclaim contract.** `POST /api/v1/storage/reclaim` with
  `{"dry_run":bool}` requires no authentication. It is storage-manager's
  over-budget backstop. It never rewrites the whole file. For an incremental
  file it returns pages for at most 30 s, then passively checkpoints the WAL.
  For a none-mode file it reports `compactionRequired: true`. The receipt
  carries `bytesBefore`, `bytesAfter`, `projectedBytesAfter`, `performed`, and
  `complete`.
- **Fenced compaction.** `POST /api/v1/storage/compact` (`agent-manager
  storage compact --reason ... --local-owner`) runs as follows:
  - It requires a human owner credential and a closed, drained maintenance
    fence.
  - It checks free space for the transient copy plus the WAL.
  - It stops the conversation indexer, which it refuses while a generation
    builds, the reconciler, and the transcript and friction schedulers.
  - On one pinned connection it sets `temp_store=FILE`, `auto_vacuum=
    INCREMENTAL`, and `VACUUM`, then `wal_checkpoint(TRUNCATE)` and
    `quick_check`.
  - It restarts the paused workers and records a receipt that
    `storage status` shows.

  The work runs asynchronously because it outlives the 3-minute HTTP write
  timeout.
- **Rowid safety.** SQLite documents that VACUUM may renumber the implicit
  rowid of a table without an `INTEGER PRIMARY KEY`.
  - **Measured.** A `VACUUM INTO` copy of the live file (made 2026-09-14 with
    the sqlite3 3.45.1 CLI) renumbered 969,404 of 986,209 legacy
    `conversation_search_documents` rows. Its FTS index, keyed by those
    rowids, then pointed at the wrong documents; the live file had 0
    mismatches. `run_events` kept every rowid, because it is still dense.
    With the modernc build Agent Manager ships, no test shape was renumbered.
  - **Legacy catalog.** Compaction refuses (HTTP 409, `ErrLegacyProjection`)
    while `conversation_search_documents` exists. The conversation-search
    projection upgrade must finish on the live file first. Its replacement,
    `conversation_search_catalog`, keys its external-content FTS by an
    explicit id.
  - **Search index.** Compaction refuses an index that is already
    inconsistent. After VACUUM it requires the FTS5 `integrity-check` to
    pass, and requires catalog and index row counts to match each other and
    the pre-compaction count.
  - **`run_events` positions.** Some consumers hold positions over
    `run_events.rowid`: `stats_checkpoint.last_rowid`,
    `cohort_watches.cursor_rowid`, and the stats engine's in-memory
    watermark. Compaction does the following:
    - It holds each one: supervision watch processing is paused and the
      stats engine lock is held.
    - It anchors each position to up to 16 stable event ids before VACUUM.
    - After VACUUM it rewrites each position from its newest surviving
      anchor, which is exact because VACUUM keeps rowid order.
    - It recomputes `event_retention_state.floor_rowid`.
    - The receipt lists every rewritten position.
  - **Safe without handling.** These uses need no rowid handling:
    - the invocation read-model watermark, which is keyed by event id
    - the conversation source snapshot, which lives only inside a build, and
      compaction refuses while a build runs
    - the imported-payload compaction cursor, which is in memory, stops with
      the reconciler, and wraps to zero
    - single-statement rowid uses
    - `incremental_vacuum`, which moves pages but never rows

Deploy order: the conversation-search projection upgrade deploys and finishes,
and `conversation_search_documents` is gone. Only then does the fenced
compaction run. After it, check
`INSERT INTO conversation_search_catalog_fts(conversation_search_catalog_fts)
VALUES('integrity-check')` and catalog/index count parity; the compaction
receipt's `searchAfter` records both.

Rehearsal on a 10.5 GB `VACUUM INTO` copy of the live file (2026-09-14):

| Step | Result |
|---|---|
| `VACUUM INTO` of the 293 GB live file (made the copy) | 199 s |
| Compaction: `VACUUM` into incremental mode | 197 s |
| Compaction: `quick_check` | 271 s, `ok` |
| Compaction: total, background writers paused | 468 s; peak RSS 40 MB (temp copy in `/var/tmp`) |
| After compaction | `auto_vacuum=incremental`, freelist 0, WAL 0 |
| Delete 2,096,129 generation staging rows | 6.3 s, freed 4.05 GB |
| Online return of those pages (one explicit reclaim, fixed 2048-page batches) | 4.03 GB in 635 s (about 6.4 MB/s at low I/O priority); file 10.37 GB → 6.32 GB, freelist 0, WAL 0 after checkpoint |

With adaptive batches, each reconciler pass on the converted copy lasted
5.1 s and returned 117-123 MB, about 23 MB/s while it ran. A multi-GB
deletion therefore returns online over tens of minutes to hours of 30-second
cycles, without a long writer hold. The
operational rule is: run bulk deletions before a fenced compaction, because
`VACUUM` cost scales with live data, not with the size of the freelist.

### Live remediation — 2026-09-14

Root causes of the 293 GB file: a full conversation-search rebuild every
15 minutes regardless of change (~3 TB/day of disk writes) into a file with
`auto_vacuum=NONE`, so every rewrite left freelist behind; event retention
that excluded imported runs; storage-manager measuring a stale class-rooted
copy instead of the lifecycle data directory; and per-run Codex homes that
re-downloaded plugin caches.

| Measure | Before | After |
|---|---|---|
| Database file | 293.49 GB (285 GB freelist) | 4.48 GB, freelist 0, `auto_vacuum=incremental` |
| Budget level (12 GiB) | alarm, 2278% | ok, 35% |
| Agent Manager disk writes | 98 MB/s | about 3 KB/s |
| Run state | 23 GB | 3.4 GB |
| Search catalog = FTS index | 986,228 = 986,228 | 933,497 = 933,497 (tool payload compaction) |

Deploy order held: the drift check first, then the projection upgrade (legacy
catalog dropped live in 195 s), then the fenced compaction (88 s, `quick_check`
ok, FTS integrity ok, 13 stored rowid positions verified). A live finding added
one fix: the incremental semantic leg counted vanished chunks of compacted tool
payloads as vector deletions, so each pass copied the whole serving vector
collection. `SemanticDocumentIDs` now limits semantic deletions to prose
documents, and the leg runs under a corpus-scaled deadline. Each prose change
still stages a full vector generation; that remains open work.

## Scope Decision

This maintenance pass removed committed startup migration/compatibility logic
and converted the local database safely. The broader schema/provider and
routed-isolation refactor is separate architecture work and was not changed in
this pass.

## Cross-References

- `storage-manager validate scenario agent-manager`
- `packages/api-core/database/schemas.go`
- `scenarios/storage-manager/docs/concepts/test-isolation-contract.md`
