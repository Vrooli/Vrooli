-- One row per issued evaluator credential. Only the SHA-256 bearer digest is
-- stored. Receipt columns remain NULL until the credential is consumed.
CREATE TABLE IF NOT EXISTS evaluator_hook_receipts (
    credential_hash BLOB NOT NULL UNIQUE CHECK (length(credential_hash) = 32),
    run_id TEXT NOT NULL,
    invocation_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    revoked_at TEXT,
    receipt_id TEXT UNIQUE,
    session_id TEXT,
    runner TEXT,
    evaluator_id TEXT,
    duration_ms INTEGER,
    occurred_at TEXT,
    CHECK (
        (consumed_at IS NULL AND receipt_id IS NULL AND session_id IS NULL AND runner IS NULL AND evaluator_id IS NULL AND duration_ms IS NULL AND occurred_at IS NULL)
        OR
        (consumed_at IS NOT NULL AND receipt_id IS NOT NULL AND session_id IS NOT NULL AND runner IS NOT NULL AND evaluator_id IS NOT NULL AND duration_ms >= 0 AND occurred_at IS NOT NULL)
    )
);

-- A run can have at most one unconsumed, unrevoked launch credential. Issue
-- revokes the prior outstanding row in the same transaction before inserting.
CREATE UNIQUE INDEX IF NOT EXISTS idx_evaluator_hook_receipts_outstanding_run
    ON evaluator_hook_receipts(run_id)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_evaluator_hook_receipts_run_time
    ON evaluator_hook_receipts(run_id, occurred_at, receipt_id)
    WHERE consumed_at IS NOT NULL;
