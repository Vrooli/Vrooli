-- Storage Manager domain tables.
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
    bytes_reclaimed INTEGER NOT NULL DEFAULT 0,
    redacted        INTEGER NOT NULL DEFAULT 0
);

-- Audit reads are always ordered oldest-first over the whole table.
CREATE INDEX IF NOT EXISTS idx_cleanup_audit_occurred_at ON cleanup_audit (occurred_at);

-- Server-owned recovery ledger. Byte quantities are integer columns so the
-- controller and read surfaces never have to parse prose or JSON to account
-- for reclaimed space.
CREATE TABLE IF NOT EXISTS recovery_runs (
    id                 TEXT PRIMARY KEY,
    started_at         TEXT NOT NULL,
    completed_at       TEXT,
    trigger            TEXT NOT NULL,
    mount              TEXT NOT NULL,
    target_free_bytes  INTEGER NOT NULL,
    reclaimed_bytes    INTEGER NOT NULL DEFAULT 0,
    result             TEXT NOT NULL,
    stopped_because    TEXT
);

CREATE INDEX IF NOT EXISTS idx_recovery_runs_started_at ON recovery_runs (started_at);

CREATE TABLE IF NOT EXISTS recovery_actions (
    id               TEXT PRIMARY KEY,
    run_id           TEXT NOT NULL REFERENCES recovery_runs(id),
    occurred_at      TEXT NOT NULL,
    provider_id      TEXT NOT NULL,
    rung             TEXT NOT NULL,
    authority        TEXT NOT NULL DEFAULT 'class',
    bytes_reclaimed  INTEGER NOT NULL DEFAULT 0,
    files_removed    INTEGER NOT NULL DEFAULT 0,
    duration_ms      INTEGER NOT NULL DEFAULT 0,
    free_before      INTEGER NOT NULL,
    free_after       INTEGER NOT NULL,
    result           TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_recovery_actions_run ON recovery_actions (run_id, occurred_at);

CREATE TABLE IF NOT EXISTS writer_snapshots (
    id               TEXT PRIMARY KEY,
    observed_at      TEXT NOT NULL,
    root             TEXT NOT NULL,
    mount            TEXT NOT NULL,
    bytes            INTEGER NOT NULL,
    delta_bytes      INTEGER NOT NULL,
    delta_hours      REAL NOT NULL,
    bytes_per_hour   INTEGER NOT NULL,
    partial          INTEGER NOT NULL DEFAULT 0,
    hot              INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_writer_snapshots_root_time ON writer_snapshots (root, observed_at);
