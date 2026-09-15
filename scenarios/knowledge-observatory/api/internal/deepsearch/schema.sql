-- deepsearch domain schema (SQLite)
-- Deep search jobs.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.

CREATE TABLE IF NOT EXISTS deep_search_jobs (
    id TEXT PRIMARY KEY,
    query TEXT NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'scenario', 'path')),
    scenario_name TEXT,
    base_path TEXT,
    max_results INTEGER,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    progress TEXT,
    results TEXT,
    agent_run_id TEXT,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_deep_search_jobs_created ON deep_search_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_deep_search_jobs_status ON deep_search_jobs(status);
