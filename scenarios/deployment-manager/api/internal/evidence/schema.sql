-- Evidence is a reference ledger. It intentionally contains no artifact bytes
-- and no producer-local filesystem paths.
CREATE TABLE IF NOT EXISTS deployment_evidence_verdicts (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    git_commit_hash TEXT NOT NULL,
    target_ramp TEXT NOT NULL,
    target_platform TEXT NOT NULL,
    target_os TEXT NOT NULL,
    device_kind INTEGER NOT NULL,
    bridge_node_id TEXT,
    bridge_job_id TEXT,
    disposition INTEGER NOT NULL,
    run_id TEXT NOT NULL,
    detail TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS deployment_evidence_verdicts_lookup
    ON deployment_evidence_verdicts(profile_id, git_commit_hash, created_at);

CREATE TABLE IF NOT EXISTS deployment_evidence_refs (
    verdict_id TEXT NOT NULL,
    producer TEXT NOT NULL,
    artifact_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    checksum TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (verdict_id) REFERENCES deployment_evidence_verdicts(id)
);

CREATE INDEX IF NOT EXISTS deployment_evidence_refs_verdict
    ON deployment_evidence_refs(verdict_id);
