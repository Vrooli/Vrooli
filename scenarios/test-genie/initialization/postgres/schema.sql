-- Database schema for Test Genie scenario.
-- Captures suite generation requests and execution metadata.

CREATE TABLE IF NOT EXISTS suite_requests (
    id UUID PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    requested_types TEXT[] NOT NULL,
    coverage_target INTEGER NOT NULL CHECK (coverage_target BETWEEN 1 AND 100),
    priority TEXT NOT NULL CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    status TEXT NOT NULL CHECK (status IN ('queued', 'delegated', 'running', 'completed', 'failed')),
    notes TEXT,
    delegation_issue_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_suite_requests_scenario
    ON suite_requests (scenario_name);

CREATE INDEX IF NOT EXISTS idx_suite_requests_status
    ON suite_requests (status);

CREATE TABLE IF NOT EXISTS suite_executions (
    id UUID PRIMARY KEY,
    suite_request_id UUID REFERENCES suite_requests(id) ON DELETE SET NULL,
    scenario_name TEXT NOT NULL,
    preset_used TEXT,
    success BOOLEAN NOT NULL,
    phases JSONB NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_suite_executions_scenario
    ON suite_executions (scenario_name);
