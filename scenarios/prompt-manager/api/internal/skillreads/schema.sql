CREATE TABLE IF NOT EXISTS skill_read_telemetry (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    observed_at INTEGER,
    observed_at_nanos INTEGER,
    payload TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_skill_read_telemetry_observed_at
    ON skill_read_telemetry(observed_at, observed_at_nanos);
