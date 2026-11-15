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


CREATE TABLE IF NOT EXISTS x402_resource_admissions (
    id TEXT PRIMARY KEY,
    price_id TEXT NOT NULL REFERENCES x402_prices(id),
    payload_digest TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('verified','settling','recovering','settled','failed','unknown','cancelled')),
    payer TEXT NOT NULL,
    transaction_id TEXT NOT NULL DEFAULT '',
    network TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'production',
    recovery_reference TEXT NOT NULL DEFAULT '',
    job_id TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    order_id TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(job_id, resource_id, order_id)
);

-- Both the legacy immediate-admit route and the deferred LPBS route share one
-- digest claim so a signed EIP-3009 authorization cannot be settled through
-- both APIs concurrently or after a retry crosses routes.
CREATE TABLE IF NOT EXISTS x402_payment_payload_claims (
    payload_digest TEXT PRIMARY KEY,
    admission_id TEXT NOT NULL,
    admission_kind TEXT NOT NULL CHECK (admission_kind IN ('inbound','resource')),
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_x402_resource_admissions_status
    ON x402_resource_admissions(status, updated_at, id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_x402_resource_admissions_transaction
    ON x402_resource_admissions(transaction_id)
    WHERE transaction_id <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_x402_inbound_admissions_transaction
    ON x402_inbound_admissions(transaction_id)
    WHERE transaction_id <> '';
