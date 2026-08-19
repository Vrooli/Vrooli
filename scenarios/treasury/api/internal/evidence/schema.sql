CREATE TABLE IF NOT EXISTS evidence_records (
    id TEXT PRIMARY KEY,
    authorization_id TEXT NOT NULL,
    mandate_id TEXT NOT NULL,
    agent_subject TEXT NOT NULL,
    verdict TEXT NOT NULL,
    violated_constraint TEXT NOT NULL,
    detail TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TRIGGER IF NOT EXISTS evidence_records_no_update
BEFORE UPDATE ON evidence_records
BEGIN SELECT RAISE(ABORT, 'evidence records are append-only'); END;

CREATE TRIGGER IF NOT EXISTS evidence_records_no_delete
BEFORE DELETE ON evidence_records
BEGIN SELECT RAISE(ABORT, 'evidence records are append-only'); END;
