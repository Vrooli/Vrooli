CREATE TABLE IF NOT EXISTS x402_prices (
    id TEXT PRIMARY KEY,
    resource_url TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    network TEXT NOT NULL,
    scheme TEXT NOT NULL CHECK (scheme = 'exact'),
    amount TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    pay_to TEXT NOT NULL,
    asset TEXT NOT NULL,
    asset_decimals INTEGER NOT NULL CHECK (asset_decimals BETWEEN 2 AND 30),
    max_timeout_seconds INTEGER NOT NULL CHECK (max_timeout_seconds > 0),
    extra_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS x402_inbound_admissions (
    id TEXT PRIMARY KEY,
    price_id TEXT NOT NULL REFERENCES x402_prices(id),
    payload_digest TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('calling','settled','failed','unknown')),
    payer TEXT NOT NULL DEFAULT '',
    transaction_id TEXT NOT NULL DEFAULT '',
    network TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_x402_inbound_admissions_price
    ON x402_inbound_admissions(price_id, created_at, id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_x402_inbound_admissions_transaction
    ON x402_inbound_admissions(transaction_id)
    WHERE transaction_id <> '';
