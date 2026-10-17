CREATE TABLE IF NOT EXISTS plan_investigation_policies (
    policy_id TEXT PRIMARY KEY,
    policy_version TEXT NOT NULL UNIQUE,
    document TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_investigation_active_policy ON plan_investigation_policies(active) WHERE active=1;

CREATE TABLE IF NOT EXISTS plan_investigation_policy_overrides (
    scope_key TEXT PRIMARY KEY,
    policy_version TEXT NOT NULL,
    document TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS plan_investigation_incidents (
    incident_id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL,
    family_id TEXT NOT NULL DEFAULT '',
    phase_id TEXT NOT NULL,
    phase_generation TEXT NOT NULL,
    policy_version TEXT NOT NULL,
    incident_fingerprint TEXT NOT NULL UNIQUE,
    mode TEXT NOT NULL,
    state TEXT NOT NULL,
    decision_json TEXT NOT NULL,
    investigation_id TEXT NOT NULL DEFAULT '',
    program_id TEXT NOT NULL DEFAULT '',
    program_status TEXT NOT NULL DEFAULT '',
    dispatch_error TEXT NOT NULL DEFAULT '',
    dispatch_claim_key TEXT NOT NULL DEFAULT '',
    dispatch_started_at TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_plan_investigation_incidents_execution ON plan_investigation_incidents(execution_id, created_at DESC);

CREATE TABLE IF NOT EXISTS plan_investigation_incident_subjects (
    incident_fingerprint TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    phase_id TEXT NOT NULL DEFAULT '',
    phase_generation TEXT NOT NULL DEFAULT '',
    relation TEXT NOT NULL DEFAULT 'implicated',
    created_at TEXT NOT NULL,
    PRIMARY KEY (incident_fingerprint, execution_id, phase_id, phase_generation),
    FOREIGN KEY (incident_fingerprint) REFERENCES plan_investigation_incidents(incident_fingerprint) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_plan_investigation_incident_subjects_execution ON plan_investigation_incident_subjects(execution_id, created_at DESC);

CREATE TABLE IF NOT EXISTS plan_investigation_occurrences (
    occurrence_id TEXT PRIMARY KEY,
    incident_fingerprint TEXT NOT NULL,
    occurrence_key TEXT NOT NULL UNIQUE,
    execution_id TEXT NOT NULL DEFAULT '',
    family_id TEXT NOT NULL DEFAULT '',
    shared_failure_ref TEXT NOT NULL DEFAULT '',
    phase_id TEXT NOT NULL DEFAULT '',
    phase_generation TEXT NOT NULL DEFAULT '',
    policy_version TEXT NOT NULL DEFAULT '',
    eligible INTEGER NOT NULL DEFAULT 0,
    decision_json TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_plan_investigation_occurrences_incident ON plan_investigation_occurrences(incident_fingerprint, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_plan_investigation_occurrences_execution ON plan_investigation_occurrences(execution_id, created_at DESC);
