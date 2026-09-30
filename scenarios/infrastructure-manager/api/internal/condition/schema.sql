CREATE TABLE IF NOT EXISTS condition_readings (
    reading_id TEXT NOT NULL,
    cell_ref TEXT NOT NULL,
    value REAL NOT NULL,
    unit TEXT NOT NULL,
    source TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    trust_verdict TEXT NOT NULL,
    PRIMARY KEY (reading_id, observed_at)
);

CREATE INDEX IF NOT EXISTS idx_condition_readings_cell_time
    ON condition_readings (cell_ref, observed_at DESC);
