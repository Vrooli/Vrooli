# Data — Source Ledger

## Authority

The SQLite journal is the one corpus authority. `entries` and its provenance
columns are append-only. A write either commits the complete source entry and
its journal high-water mark or commits nothing. Corrections append facet
assignments; they do not rewrite the entry body. The production authority is
`scenarios/source-ledger/data/source-ledger.db`; `vrooli-memory` retains only
the harness integration tables after the phase-15 cutover.

The database is non-regenerable and must be registered with
`data-backup-manager`. A restore is valid only when entry counts and per-row
body hashes match the live authority.

## Derived Data

The forest is a rebuildable cache. `summaries`, `tree_edges`, embeddings,
facet text, and recall statistics may be regenerated from journal rows and
policy data. Compaction never deletes a source entry. A provider outage may
pause enrichment, but it cannot make the journal unavailable for append or
projection consumers.

## Scope Model

Every corpus-bearing table carries `scope` or is joined through an owning
scope. `agent-memory` is the default only at the request boundary. A named
scope has its own vocabulary, retention policy, residency budget, wake budget,
and frontier target. Queries must filter scope before scoring or rendering.

## Migration Invariants

An append-only migration must preserve:

1. the union of entry IDs from both source sets;
2. each shared entry body hash;
3. dependent facet text, embedding, assignment, mark, and provenance rows;
4. the high-water mark and append-only triggers; and
5. the distinction between journal authority and rebuildable derived rows.

## Retention

Retention controls recall and compaction eligibility. It never authorizes
deletion of journal rows. Pins, reviews, and supersession are append-safe
state transitions over immutable source material.

## Architecture Maturity

The data contract distinguishes authoritative journal rows from rebuildable
forest state. Backup registration and migration evidence are deliberately
deferred to the corpus-authority phase.

## Contracts And Data Flow

Append requests write the journal authority first; derived forest and recall
state are downstream projections that can be rebuilt from authoritative data.


This document distinguishes durable source material from derived ledger state.

## Storage Overview

Source Ledger owns two SQLite files. The data-class `source-ledger.db`
holds the authoritative journal and its derived indexes. The cache-class
`files-index.db` holds FTS chunks and embeddings derived from the research
home. The manifest declares each file and both SQLite sidecars. The recall
benchmark reads a committed source snapshot and uses disposable copies
for synthetic entries.

Both writers initialize incremental auto-vacuum before creating schema in a
new file. Every 15 minutes the owner runs a bounded maintenance pass on each
file, measuring the freelist and the database and WAL lengths. It returns free
pages above the shared threshold while keeping a reserve for reuse. SQLite
checkpoints the WAL, and the maintenance pass reports contention as a
deferral. Journal retention and compaction govern records separately.

The journal stores append positions as explicit integer keys. On the first
boot of an older database, one transaction copies each existing rowid to a
durable position, including gaps, and validates the original high-water
mark. Subsequent appends write the entry, position, and high-water mark
together. Cursor lookups and ties between entries with the same timestamp
use those positions. The migration preserves the entry table and its
relationships in place.

Legacy journal files keep their current auto-vacuum mode. The storage
maintenance pilot rehearses migration and a full VACUUM on an isolated
journal fixture. Any future conversion of the live journal requires an
owner-controlled stopped window and verification against a copy of that
database.

For an existing research index, use the owner conversion command from
`scenarios/source-ledger/api` after the lifecycle reports the live instance
stopped:

```bash
go run ./cmd/convert-files-index --apply
```

The command holds the lifecycle lock, verifies the known FTS schema and its
stable chunk IDs, and checks free space before rewriting. Its JSON receipt
reports the mode, file sizes, and any deferral. A second run verifies the
existing incremental mode and handles any pending checkpoint.

## Data Ownership

Journal owns source entries; forest, facets, recall, and federation own only
their derived or policy records.

## Schema Map

The schema map follows the ledger domains named in `DOMAINS.md`; generated
proto packages will become the wire-level schema source of truth.

## Migrations And Compatibility

Migration must preserve entry identity, body hashes, provenance, and append-only
guards before any consumer cutover. The one-shot copy of the engine-owned
tables from `vrooli-memory` ran on 2026-08-07 (PROGRESS.md, phase 15) and its
command was then retired; `harness_projections`, `harness_import_runs` and
`journal_retry_queue` stayed in `vrooli-memory`. The source-ledger database is
a non-regenerable `data-backup-manager` target.

## Import / Export

Import and export are bounded service operations and never authorize edits to
existing journal rows.

## Retention And Deletion

Retention affects recall and compaction eligibility; the journal has no delete
path.

## Privacy Notes

Source bodies and provenance remain inside the owning ledger database and are
not copied into consumer projections.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DOMAINS.md`](DOMAINS.md)
- [`FLOWS.md`](FLOWS.md)
