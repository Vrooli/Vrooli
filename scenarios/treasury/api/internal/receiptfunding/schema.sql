CREATE TABLE IF NOT EXISTS receipt_funding_deliveries (
    id TEXT PRIMARY KEY,
    operator_identity TEXT NOT NULL,
    wallet_id TEXT NOT NULL,
    budget_id TEXT NOT NULL,
    source_check_id TEXT NOT NULL,
    request_proto BLOB NOT NULL,
    request_sha256 TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('queued','accepted')),
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    accepted_proto BLOB,
    created_at TEXT NOT NULL,
    accepted_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS receipt_funding_credit_lot_links (
    delivery_id TEXT PRIMARY KEY REFERENCES receipt_funding_deliveries(id),
    receipt_id TEXT NOT NULL,
    credit_lot_id TEXT NOT NULL,
    credit_lot_owner TEXT NOT NULL,
    linked_at TEXT NOT NULL,
    UNIQUE(credit_lot_owner, credit_lot_id),
    UNIQUE(receipt_id, credit_lot_owner)
);
CREATE INDEX IF NOT EXISTS idx_receipt_funding_pending ON receipt_funding_deliveries(status,created_at,id);
CREATE INDEX IF NOT EXISTS idx_receipt_funding_wallet ON receipt_funding_deliveries(operator_identity,wallet_id,created_at,id);
