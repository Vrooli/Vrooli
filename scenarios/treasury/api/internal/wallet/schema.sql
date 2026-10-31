CREATE TABLE IF NOT EXISTS research_wallets (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL,
    operator_identity TEXT NOT NULL,
    network TEXT NOT NULL CHECK (network IN ('eip155:8453', 'eip155:84532')),
    address TEXT NOT NULL DEFAULT '',
    public_key TEXT NOT NULL DEFAULT '',
    custody_reference TEXT NOT NULL UNIQUE,
    key_minted INTEGER NOT NULL DEFAULT 0,
    recovery_minted INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    recovery_status TEXT NOT NULL DEFAULT 'not_qualified',
    recovery_sha256 TEXT NOT NULL DEFAULT ''
);
