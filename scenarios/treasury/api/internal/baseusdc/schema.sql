CREATE TABLE IF NOT EXISTS baseusdc_receipt_checks (
    id TEXT PRIMARY KEY,
    operator_identity TEXT NOT NULL,
    wallet_id TEXT NOT NULL,
    network TEXT NOT NULL,
    transaction_hash TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    outcome TEXT NOT NULL,
    error_code TEXT NOT NULL DEFAULT '',
    checked_at TEXT NOT NULL,
    transfers_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_baseusdc_receipt_checks_lookup
    ON baseusdc_receipt_checks(operator_identity, wallet_id, transaction_hash, checked_at DESC);
