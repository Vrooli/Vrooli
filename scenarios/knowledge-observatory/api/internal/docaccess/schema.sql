-- docaccess domain schema (SQLite)
-- Document access log for reads, writes, and resets.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.

CREATE TABLE IF NOT EXISTS doc_access_log (
    id TEXT PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    doc_type TEXT NOT NULL,
    operation TEXT NOT NULL CHECK (operation IN ('read', 'write', 'reset')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doc_access_log_scenario ON doc_access_log(scenario_name);
CREATE INDEX IF NOT EXISTS idx_doc_access_log_created ON doc_access_log(created_at DESC);
