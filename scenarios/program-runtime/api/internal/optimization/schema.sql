CREATE TABLE IF NOT EXISTS optimization_runs (
    run_id TEXT PRIMARY KEY,
    revision INTEGER NOT NULL,
    stage TEXT NOT NULL,
    record_json TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_optimization_runs_stage_updated
    ON optimization_runs(stage, updated_at);
