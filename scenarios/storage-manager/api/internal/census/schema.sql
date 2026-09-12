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

CREATE TABLE IF NOT EXISTS census_snapshot_metrics (
    snapshot_id TEXT PRIMARY KEY,
    drift_bytes INTEGER NOT NULL DEFAULT 0,
    growth_slope_bytes_per_hour REAL
);

-- One narrow row per declared entry makes trend queries independent of the
-- immutable report blob size. The report remains the evidence record; this is
-- the read model for growth ranking and projections.
CREATE TABLE IF NOT EXISTS census_entry_samples (
    snapshot_id  TEXT NOT NULL,
    observed_at  TEXT NOT NULL,
    root         TEXT NOT NULL,
    owner_kind   TEXT NOT NULL DEFAULT '',
    owner_id     TEXT NOT NULL,
    entry_name   TEXT NOT NULL,
    bytes        INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (snapshot_id, owner_kind, owner_id, entry_name)
);
CREATE INDEX IF NOT EXISTS idx_census_entry_samples_root_time
    ON census_entry_samples (root, observed_at DESC, owner_kind, owner_id, entry_name);
CREATE INDEX IF NOT EXISTS idx_census_entry_samples_time
    ON census_entry_samples (observed_at);
