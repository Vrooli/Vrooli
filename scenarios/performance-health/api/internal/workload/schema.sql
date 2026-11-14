CREATE TABLE IF NOT EXISTS performance_workload_receipts (
    operation_id TEXT PRIMARY KEY,
    scenario TEXT NOT NULL,
    workload TEXT NOT NULL,
    started_at_ns INTEGER NOT NULL,
    completed INTEGER NOT NULL DEFAULT 0,
    reading_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS performance_workload_latest
    ON performance_workload_receipts (scenario, workload, started_at_ns DESC);
