CREATE TABLE IF NOT EXISTS investigation_runs (
    id TEXT PRIMARY KEY,
    entry_id TEXT NOT NULL,
    execution_mode TEXT NOT NULL,
    status TEXT NOT NULL,
    skip_reason TEXT NOT NULL DEFAULT '',
    exit_code INTEGER NOT NULL DEFAULT 0,
    timed_out INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    completed_at TEXT NOT NULL,
    duration_seconds REAL NOT NULL DEFAULT 0,
    host_os TEXT NOT NULL,
    host_arch TEXT NOT NULL,
    result_json TEXT NOT NULL DEFAULT '',
    stderr_tail TEXT NOT NULL DEFAULT '',
    anomaly_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_investigation_runs_started ON investigation_runs(started_at);
CREATE INDEX IF NOT EXISTS idx_investigation_runs_entry_started ON investigation_runs(entry_id, started_at);
CREATE INDEX IF NOT EXISTS idx_investigation_runs_status ON investigation_runs(status);

CREATE TABLE IF NOT EXISTS investigation_findings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL REFERENCES investigation_runs(id) ON DELETE CASCADE,
    severity TEXT NOT NULL,
    code TEXT NOT NULL,
    summary TEXT NOT NULL,
    detail_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_investigation_findings_run ON investigation_findings(run_id);
