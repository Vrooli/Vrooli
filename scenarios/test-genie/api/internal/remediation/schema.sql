CREATE TABLE IF NOT EXISTS remediation_jobs (
    id TEXT PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    status TEXT NOT NULL,
    source_json TEXT NOT NULL,
    selected_finding_ids_json TEXT NOT NULL,
    additional_context TEXT NOT NULL DEFAULT '',
    attribution_json TEXT NOT NULL DEFAULT '{}',
    verification_json TEXT NOT NULL DEFAULT '{}',
    failure TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    cancelled_at TEXT
);
CREATE INDEX IF NOT EXISTS remediation_jobs_scenario_created_idx ON remediation_jobs(scenario_name, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS remediation_jobs_one_active_per_scenario
    ON remediation_jobs(scenario_name)
    WHERE status IN ('created', 'running', 'agent_completed', 'verification_running');
