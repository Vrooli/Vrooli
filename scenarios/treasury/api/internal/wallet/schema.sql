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

-- Revocation is separate from the immutable wallet row so existing Treasury
-- databases can gain durable key lifecycle state without rewriting custody
-- bindings or weakening their mint witnesses.
CREATE TABLE IF NOT EXISTS research_wallet_revocations (
    wallet_id TEXT PRIMARY KEY REFERENCES research_wallets(id),
    revoked_at TEXT NOT NULL,
    successor_wallet_id TEXT NOT NULL UNIQUE REFERENCES research_wallets(id)
);

-- An intent serializes concurrent rotation requests and makes retries after a
-- process restart converge on the same fresh successor ID.
CREATE TABLE IF NOT EXISTS research_wallet_rotation_intents (
    wallet_id TEXT PRIMARY KEY REFERENCES research_wallets(id),
    successor_wallet_id TEXT NOT NULL UNIQUE,
    started_at TEXT NOT NULL
);
