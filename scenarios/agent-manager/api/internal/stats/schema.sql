-- ============================================================================
-- Stats engine checkpoint (Phase 3)
-- ============================================================================
-- stats_checkpoint persists the in-memory engine's last-processed
-- run_events.rowid so a crash or process restart resumes from the saved
-- watermark instead of replaying every event from zero. The "name"
-- column allows multiple engines (e.g., a future per-tenant view) to
-- share the table without colliding.
CREATE TABLE IF NOT EXISTS stats_checkpoint (
    name TEXT PRIMARY KEY,
    last_rowid INTEGER NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

