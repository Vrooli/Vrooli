-- SQLite schema for Test Genie operational storage.
-- Stores queued suite requests and persisted suite executions.

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

CREATE TABLE IF NOT EXISTS suite_executions (
    id TEXT PRIMARY KEY,
    suite_request_id TEXT REFERENCES suite_requests(id) ON DELETE SET NULL,
    scenario_name TEXT NOT NULL,
    preset_used TEXT,
    requested_preset TEXT,
    requested_phases TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(requested_phases)),
    requested_skip_phases TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(requested_skip_phases)),
    planned_phases TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(planned_phases)),
    fail_fast INTEGER NOT NULL DEFAULT 0 CHECK (fail_fast IN (0, 1)),
    success INTEGER NOT NULL CHECK (success IN (0, 1)),
    phases TEXT NOT NULL CHECK (json_valid(phases)),
    started_at TEXT NOT NULL,
    completed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_suite_executions_scenario
    ON suite_executions (scenario_name);

CREATE INDEX IF NOT EXISTS idx_suite_executions_completed_at
    ON suite_executions (completed_at DESC);
