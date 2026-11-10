CREATE TABLE IF NOT EXISTS budget_wallets (
 budget_id TEXT PRIMARY KEY REFERENCES budgets(id),
 wallet_id TEXT NOT NULL UNIQUE REFERENCES research_wallets(id),
 parent_wallet_id TEXT NOT NULL REFERENCES research_wallets(id),
 operator_identity TEXT NOT NULL,
 network TEXT NOT NULL,
 agent_subject TEXT NOT NULL,
 agent_run_id TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS funding_intents (
 id TEXT PRIMARY KEY,
 budget_id TEXT NOT NULL REFERENCES budget_wallets(budget_id),
 operator_identity TEXT NOT NULL,
 atomic_amount TEXT NOT NULL,
 request_proto BLOB NOT NULL,
 start_block TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('waiting','observed','delivering','credited')),
 transaction_hash TEXT NOT NULL DEFAULT '',
 receipt_delivery_id TEXT NOT NULL DEFAULT '',
 notification_state TEXT NOT NULL DEFAULT 'pending',
 last_error TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS funding_intents_one_active_budget ON funding_intents(budget_id) WHERE state<>'credited';
CREATE TABLE IF NOT EXISTS withdrawal_destinations (
 budget_id TEXT NOT NULL REFERENCES budget_wallets(budget_id),
 address TEXT NOT NULL,
 label TEXT NOT NULL,
 operator_identity TEXT NOT NULL,
 created_at TEXT NOT NULL,
 PRIMARY KEY(budget_id,address)
);
CREATE TABLE IF NOT EXISTS withdrawal_quotes (
 id TEXT PRIMARY KEY,
 budget_id TEXT NOT NULL REFERENCES budget_wallets(budget_id),
 operator_identity TEXT NOT NULL,
 public_proto BLOB NOT NULL,
 execution_quote TEXT NOT NULL,
 expires_at TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS withdrawals (
 id TEXT PRIMARY KEY,
 quote_id TEXT NOT NULL UNIQUE REFERENCES withdrawal_quotes(id),
 operator_identity TEXT NOT NULL,
 idempotency_key TEXT NOT NULL UNIQUE,
 authorization_id TEXT NOT NULL,
 settlement_id TEXT NOT NULL,
 state TEXT NOT NULL,
 transaction_hash TEXT NOT NULL DEFAULT '',
 last_error TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS mandate_session_keys (
    id TEXT PRIMARY KEY,
    mandate_id TEXT NOT NULL,
    budget_id TEXT NOT NULL,
    wallet_id TEXT NOT NULL UNIQUE,
    operator_identity TEXT NOT NULL,
    agent_subject TEXT NOT NULL,
    agent_run_id TEXT NOT NULL,
    network TEXT NOT NULL,
    generation INTEGER NOT NULL CHECK (generation > 0),
    state TEXT NOT NULL CHECK (state IN ('active','rotated','revoked')),
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(mandate_id, generation)
);
CREATE UNIQUE INDEX IF NOT EXISTS mandate_session_one_active
    ON mandate_session_keys(mandate_id) WHERE state='active';
CREATE INDEX IF NOT EXISTS mandate_session_agent_read
    ON mandate_session_keys(budget_id, agent_subject, agent_run_id, state);

-- The session proof key is generated and held by the agent run. Treasury keeps
-- only its public key; the funded budget EOA remains under the treasury signer.
CREATE TABLE IF NOT EXISTS mandate_agent_session_keys (
    mandate_id TEXT NOT NULL,
    generation INTEGER NOT NULL,
    budget_id TEXT NOT NULL,
    agent_subject TEXT NOT NULL,
    agent_run_id TEXT NOT NULL,
    public_key TEXT NOT NULL,
    registered_at TEXT NOT NULL,
    PRIMARY KEY(mandate_id,generation),
    FOREIGN KEY(mandate_id,generation) REFERENCES mandate_session_keys(mandate_id,generation)
);

-- A proof binds one approved authorization to one settlement request. Identical
-- retries survive process restart; any changed settlement tuple is a replay.
CREATE TABLE IF NOT EXISTS mandate_agent_session_proofs (
    authorization_id TEXT PRIMARY KEY,
    mandate_id TEXT NOT NULL,
    generation INTEGER NOT NULL,
    intent_digest TEXT NOT NULL,
    signature_digest TEXT NOT NULL,
    settlement_id TEXT NOT NULL,
    instrument_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    consumed_at TEXT NOT NULL,
    FOREIGN KEY(mandate_id,generation) REFERENCES mandate_agent_session_keys(mandate_id,generation)
);

-- A successful session authorization consumes its authorization id before
-- custody signs. A lost response is reconciled from the durable settlement
-- fence and protected transaction payload, never by signing a second payload.
CREATE TABLE IF NOT EXISTS mandate_session_signatures (
    authorization_id TEXT PRIMARY KEY,
    mandate_id TEXT NOT NULL,
    wallet_id TEXT NOT NULL,
    transaction_digest TEXT NOT NULL,
    consumed_at TEXT NOT NULL
);
