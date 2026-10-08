
CREATE TABLE IF NOT EXISTS runner_health_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    runner_type TEXT NOT NULL,
    status TEXT NOT NULL,        -- ok | unknown | failed
    reason TEXT,                 -- nullable; populated on failed (fallback.Reason)
    message TEXT,                -- nullable
    triggered_by TEXT NOT NULL   -- run_id or "probe"
);
CREATE INDEX IF NOT EXISTS idx_rha_runner ON runner_health_audit(runner_type, timestamp DESC);
