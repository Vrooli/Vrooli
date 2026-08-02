-- Cleanup Manager domain tables.
--
-- Only durable state lives here. Plans and apply reports are deliberately
-- absent: a plan is a measurement of the filesystem at one instant, consumed
-- within seconds and meaningless after a restart, and one plan on a real host
-- serialises to several megabytes. Persisting those in the scenario whose job
-- is reclaiming disk space would be self-defeating. See
-- internal/orchestrator/sqlite_store.go.

-- The single active cleanup policy.
--
-- One row, fixed id. The operator's chosen profile is a durable decision: while
-- it lived only in process memory it silently reverted to the shipped default
-- on every restart, which is indistinguishable from cleanup never having been
-- enabled.
CREATE TABLE IF NOT EXISTS cleanup_policy (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    payload    TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- The audit trail of what cleanup observed, planned, and deleted.
--
-- Evidence of a deletion is worthless if it disappears with the process that
-- performed the deletion, so this outlives the run that wrote it. Messages are
-- redacted of filesystem paths before insertion (cleanup.Redact); the redacted
-- column records whether that happened.
CREATE TABLE IF NOT EXISTS cleanup_audit (
    id              TEXT PRIMARY KEY,
    occurred_at     TEXT NOT NULL,
    type            TEXT NOT NULL,
    plan_id         TEXT,
    provider_id     TEXT,
    idempotency_key TEXT,
    message         TEXT,
    redacted        INTEGER NOT NULL DEFAULT 0
);

-- Audit reads are always ordered oldest-first over the whole table.
CREATE INDEX IF NOT EXISTS idx_cleanup_audit_occurred_at ON cleanup_audit (occurred_at);
