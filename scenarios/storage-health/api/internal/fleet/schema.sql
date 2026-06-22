-- fleet domain schema: the latest persisted fleet storage-inventory snapshot.
--
-- storage-health dogfoods the very conventions it enforces: this is a
-- per-domain, embedded, idempotent (CREATE … IF NOT EXISTS) schema with no
-- ALTER statements. One row per scenario; the snapshot is replaced wholesale on
-- each ScanFleet (delete-all + insert within a transaction), and aggregates are
-- recomputed from the rows on read — so there is no denormalized total to drift.
CREATE TABLE IF NOT EXISTS fleet_entries (
    scenario          TEXT PRIMARY KEY,
    engines           TEXT    NOT NULL DEFAULT '',
    primary_engine    TEXT    NOT NULL DEFAULT '',
    language          TEXT    NOT NULL DEFAULT '',
    storage_stage     TEXT    NOT NULL DEFAULT '',
    isolation_ready   INTEGER NOT NULL DEFAULT 0,
    isolation_reason  TEXT    NOT NULL DEFAULT '',
    namespace_adopted INTEGER NOT NULL DEFAULT 0,
    has_backup_target INTEGER NOT NULL DEFAULT 0,
    finding_count     INTEGER NOT NULL DEFAULT 0,
    error_count       INTEGER NOT NULL DEFAULT 0,
    autofixable_count INTEGER NOT NULL DEFAULT 0,
    scanned_at        TEXT    NOT NULL DEFAULT ''
);
