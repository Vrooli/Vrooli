-- dochealing domain schema (SQLite)
-- Documentation healing jobs.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.

CREATE TABLE IF NOT EXISTS doc_heal_jobs (
    id TEXT PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    issues TEXT,
    auto_approve BOOLEAN DEFAULT FALSE,
    dry_run BOOLEAN DEFAULT FALSE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'needs_review', 'approved', 'rejected', 'failed')),
    progress TEXT,
    diff TEXT,
    agent_run_id TEXT,
    error TEXT,
    health_before REAL,
    health_after REAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    approved_at TIMESTAMP,
    approved_by TEXT,
    rejected_at TIMESTAMP,
    rejected_by TEXT,
    reject_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_doc_heal_jobs_created ON doc_heal_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_doc_heal_jobs_status ON doc_heal_jobs(status);
CREATE INDEX IF NOT EXISTS idx_doc_heal_jobs_scenario ON doc_heal_jobs(scenario_name);
