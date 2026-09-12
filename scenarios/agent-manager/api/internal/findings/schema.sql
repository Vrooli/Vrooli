CREATE TABLE IF NOT EXISTS run_findings (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    investigation_run_id TEXT NOT NULL,
    category TEXT NOT NULL,
    severity TEXT NOT NULL,
    recommendation_text TEXT NOT NULL,
    evidence TEXT NOT NULL DEFAULT '',
    target_path TEXT NOT NULL DEFAULT '',
    fingerprint TEXT NOT NULL,
    operator_decision TEXT NOT NULL DEFAULT '',
    target_measure TEXT NOT NULL DEFAULT 'finding-recurrence-rate',
    before_value REAL,
    after_value REAL,
    effectiveness TEXT NOT NULL DEFAULT 'not_yet_measurable',
    friction_topic TEXT NOT NULL DEFAULT '',
    cites_resolved_commands INTEGER NOT NULL DEFAULT 0,
    cites_real_outcome INTEGER NOT NULL DEFAULT 0,
    cites_attributed_owner INTEGER NOT NULL DEFAULT 0,
    quality_signal TEXT NOT NULL DEFAULT 'unavailable',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_run_findings_fingerprint ON run_findings(fingerprint, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_run_findings_investigation ON run_findings(investigation_run_id);
CREATE INDEX IF NOT EXISTS idx_run_findings_created ON run_findings(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_run_findings_investigation_fingerprint ON run_findings(investigation_run_id, category, fingerprint);

-- A finding may implicate one or more subject runs, but a cohort member is
-- never implicated merely because it was included in the investigation.
-- Keep those edges separate from the recurrence row's legacy run_id column.
CREATE TABLE IF NOT EXISTS run_finding_subject_edges (
    finding_id TEXT NOT NULL,
    subject_run_id TEXT NOT NULL,
    relation TEXT NOT NULL,
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    PRIMARY KEY (finding_id, subject_run_id, relation),
    FOREIGN KEY (finding_id) REFERENCES run_findings(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_run_finding_subject_edges_subject ON run_finding_subject_edges(subject_run_id, created_at DESC);
