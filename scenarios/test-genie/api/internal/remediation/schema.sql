CREATE TABLE IF NOT EXISTS remediation_jobs (
    id TEXT PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    target_kind TEXT NOT NULL DEFAULT 'scenario',
    target_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    source_json TEXT NOT NULL,
	 source_hash TEXT NOT NULL DEFAULT '',
    selected_finding_ids_json TEXT NOT NULL,
    selected_requirement_ids_json TEXT NOT NULL DEFAULT '[]',
    selection_hash TEXT NOT NULL DEFAULT '',
	launch_attempt INTEGER NOT NULL DEFAULT 0,
    additional_context TEXT NOT NULL DEFAULT '',
    attribution_json TEXT NOT NULL DEFAULT '{}',
    verification_json TEXT NOT NULL DEFAULT '{}',
    failure TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    cancelled_at TEXT
);
CREATE INDEX IF NOT EXISTS remediation_jobs_scenario_created_idx ON remediation_jobs(scenario_name, created_at DESC);
CREATE TABLE IF NOT EXISTS remediation_attempts (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    state TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    role_ref TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    run_id TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS remediation_attempts_job_created_idx
    ON remediation_attempts(job_id, created_at ASC);
