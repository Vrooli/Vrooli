CREATE TABLE IF NOT EXISTS utilization_history_samples (
    id TEXT PRIMARY KEY,
    observed_at TEXT NOT NULL,
    physical_complete INTEGER NOT NULL,
    provider_run_ids_json TEXT NOT NULL,
    unmapped_run_ids_json TEXT NOT NULL,
    unknown_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_utilization_history_observed
    ON utilization_history_samples(observed_at);
