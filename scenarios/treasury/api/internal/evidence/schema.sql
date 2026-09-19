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

-- One terminal, self-contained replay record per spend attempt. Decision
-- evidence above remains the chronological policy trace; this table is the
-- immutable financial record that can reconstruct an attempt without joining
-- mutable authorization, approval, instrument, or settlement state.
CREATE TABLE IF NOT EXISTS spend_attempt_evidence (
    id TEXT PRIMARY KEY,
    authorization_id TEXT NOT NULL UNIQUE,
    mandate_id TEXT NOT NULL,
    approval_id TEXT NOT NULL DEFAULT '',
    settlement_id TEXT NOT NULL DEFAULT '',
    instrument_id TEXT NOT NULL DEFAULT '',
    agent_subject TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('refused','declined','expired','failed','settled')),
    basis TEXT NOT NULL,
    request_json TEXT NOT NULL CHECK (json_valid(request_json)),
    rail_response_json TEXT NOT NULL CHECK (json_valid(rail_response_json)),
    receipt_json TEXT NOT NULL CHECK (json_valid(receipt_json)),
    recorded_at TEXT NOT NULL,
    retain_until TEXT NOT NULL
);

CREATE TRIGGER IF NOT EXISTS spend_attempt_evidence_no_update
BEFORE UPDATE ON spend_attempt_evidence
BEGIN SELECT RAISE(ABORT, 'spend attempt evidence is append-only'); END;

CREATE TRIGGER IF NOT EXISTS spend_attempt_evidence_no_delete
BEFORE DELETE ON spend_attempt_evidence
BEGIN SELECT RAISE(ABORT, 'spend attempt evidence is append-only'); END;
