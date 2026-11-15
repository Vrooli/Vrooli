CREATE TABLE IF NOT EXISTS skill_read_telemetry (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    observed_at INTEGER,
    observed_at_nanos INTEGER,
    payload TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_skill_read_telemetry_observed_at
    ON skill_read_telemetry(observed_at, observed_at_nanos);

-- One row per (verified agent run, skill). It answers "has this run read this
-- skill?" for read reminders. It is separate from skill_read_telemetry because
-- that log is capped by row count, and one busy service caller can turn it
-- over within an hour; a receipt must outlive that churn.
CREATE TABLE IF NOT EXISTS skill_run_receipts (
    run_id TEXT NOT NULL,
    skill_id TEXT NOT NULL,
    first_read_at INTEGER NOT NULL,
    last_read_at INTEGER NOT NULL,
    read_count INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (run_id, skill_id)
);
CREATE INDEX IF NOT EXISTS idx_skill_run_receipts_last_read_at
    ON skill_run_receipts(last_read_at);
