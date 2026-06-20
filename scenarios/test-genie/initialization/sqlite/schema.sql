-- SQLite schema for Test Genie operational storage.
-- Stores queued suite requests. The persisted suite_executions table is owned
-- by the execution domain (internal/execution/schema.sql) and applied via the
-- per-domain schema registry — it is intentionally NOT defined here.

CREATE TABLE IF NOT EXISTS suite_requests (
    id TEXT PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    requested_types TEXT NOT NULL CHECK (json_valid(requested_types)),
    coverage_target INTEGER NOT NULL CHECK (coverage_target BETWEEN 1 AND 100),
    priority TEXT NOT NULL CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    status TEXT NOT NULL CHECK (status IN ('queued', 'delegated', 'running', 'completed', 'failed')),
    notes TEXT,
    delegation_issue_id TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_suite_requests_scenario
    ON suite_requests (scenario_name);

CREATE INDEX IF NOT EXISTS idx_suite_requests_status
    ON suite_requests (status);

CREATE INDEX IF NOT EXISTS idx_suite_requests_updated_at
    ON suite_requests (updated_at DESC);
