CREATE TABLE IF NOT EXISTS optimization_runs (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    scoring_profile TEXT NOT NULL,
    baseline_snapshot_id TEXT NOT NULL,
    recommendation TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_optimization_runs_status
ON optimization_runs(status, updated_at DESC);

CREATE TABLE IF NOT EXISTS optimization_candidates (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL,
    score REAL NOT NULL,
    evidence_json TEXT NOT NULL,
    approval_required INTEGER NOT NULL,
    rollback_supported INTEGER NOT NULL,
    rollback_handle TEXT NOT NULL,
    baseline_snapshot_id TEXT NOT NULL,
    candidate_snapshot_id TEXT NOT NULL,
    after_snapshot_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_optimization_candidates_run
ON optimization_candidates(run_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS optimization_approval_records (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    candidate_id TEXT NOT NULL,
    approved INTEGER NOT NULL,
    note TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_optimization_approvals_run
ON optimization_approval_records(run_id, created_at DESC);

CREATE TABLE IF NOT EXISTS optimization_rollback_records (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    candidate_id TEXT NOT NULL,
    status TEXT NOT NULL,
    details_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_optimization_rollbacks_run
ON optimization_rollback_records(run_id, created_at DESC);
