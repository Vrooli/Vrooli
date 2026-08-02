-- Immutable storage-manager census observations. The report JSON is the
-- forward-compatible record; indexed totals support trend queries.
CREATE TABLE IF NOT EXISTS census_snapshots (
    id                TEXT PRIMARY KEY,
    observed_at       TEXT NOT NULL,
    root              TEXT NOT NULL,
    measured_bytes    INTEGER NOT NULL DEFAULT 0,
    attributed_bytes  INTEGER NOT NULL DEFAULT 0,
    unattributed_bytes INTEGER NOT NULL DEFAULT 0,
    confidence        TEXT NOT NULL DEFAULT 'degraded',
    report_json       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_census_snapshots_root_time
    ON census_snapshots (root, observed_at DESC);
