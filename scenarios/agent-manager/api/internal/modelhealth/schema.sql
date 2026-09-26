CREATE TABLE IF NOT EXISTS model_health_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    runner_type TEXT NOT NULL,
    model_id TEXT NOT NULL,
    status TEXT NOT NULL,        -- ok | unknown | failed
    reason TEXT,                 -- nullable; populated on failed (fallback.Reason)
    message TEXT,                -- nullable; short operator-readable summary
    triggered_by TEXT NOT NULL   -- run_id or "probe"
);
CREATE INDEX IF NOT EXISTS idx_mha_runner_model ON model_health_audit(runner_type, model_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_mha_timestamp ON model_health_audit(timestamp DESC);

