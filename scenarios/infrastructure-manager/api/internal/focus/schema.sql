CREATE TABLE IF NOT EXISTS focus_findings (
    finding_id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    cell_ref TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    sensor_ref TEXT NOT NULL,
    expected_return TEXT NOT NULL,
    rank INTEGER NOT NULL,
    rank_explanation TEXT NOT NULL,
    observed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS focus_efficacy (
    finding_id TEXT NOT NULL,
    sensor_ref TEXT NOT NULL,
    expected_return TEXT NOT NULL,
    observed_return TEXT NOT NULL,
    verdict TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    PRIMARY KEY (finding_id, observed_at)
);

CREATE INDEX IF NOT EXISTS idx_focus_efficacy_finding_time
    ON focus_efficacy (finding_id, observed_at DESC);
