CREATE TABLE IF NOT EXISTS earning_submissions (
    id TEXT PRIMARY KEY,
    adapter_identity TEXT NOT NULL CHECK (length(trim(adapter_identity)) > 0),
    dedup_key TEXT NOT NULL CHECK (length(trim(dedup_key)) > 0),
    payload_summary TEXT NOT NULL CHECK (length(trim(payload_summary)) > 0),
    grant_id TEXT NOT NULL CHECK (length(trim(grant_id)) > 0),
    actor_identity TEXT NOT NULL CHECK (length(trim(actor_identity)) > 0),
    submitted_at TEXT NOT NULL,
    UNIQUE (adapter_identity, dedup_key)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_earning_submissions_grant
    ON earning_submissions(grant_id);
