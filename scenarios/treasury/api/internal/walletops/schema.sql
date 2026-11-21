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
CREATE TABLE IF NOT EXISTS funding_intent_watch_cursors (
 intent_id TEXT PRIMARY KEY REFERENCES funding_intents(id),
 next_block TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
-- The first exact, finalized native-USDC transfer is bound to one intent before
-- Treasury queues any Money Ledger side effect. Retries reuse this check/log
-- and stable delivery ID; a receipt log cannot fund two different intents.
CREATE TABLE IF NOT EXISTS funding_intent_receipt_bindings (
 intent_id TEXT PRIMARY KEY REFERENCES funding_intents(id),
 operator_identity TEXT NOT NULL,
 wallet_id TEXT NOT NULL,
 network TEXT NOT NULL,
 asset_id TEXT NOT NULL,
 transaction_hash TEXT NOT NULL,
 log_index INTEGER NOT NULL CHECK(log_index >= 0),
 atomic_amount TEXT NOT NULL,
 recipient TEXT NOT NULL,
 block_hash TEXT NOT NULL,
 check_id TEXT NOT NULL,
 delivery_id TEXT NOT NULL UNIQUE,
 created_at TEXT NOT NULL,
 UNIQUE(wallet_id,network,asset_id,transaction_hash,log_index)
);
-- Reserve each chain log globally, even if a malformed or replayed verifier
-- result presents it as funding a different budget wallet. The conflict table
-- preserves every legacy row if earlier schema versions bound one log more
-- than once; the trigger below then fails schema migration closed.
CREATE TABLE IF NOT EXISTS funding_intent_receipt_log_claims (
 network TEXT NOT NULL,
 asset_id TEXT NOT NULL,
 transaction_hash TEXT NOT NULL,
 log_index INTEGER NOT NULL CHECK(log_index >= 0),
 intent_id TEXT NOT NULL UNIQUE REFERENCES funding_intents(id),
 created_at TEXT NOT NULL,
 PRIMARY KEY(network,asset_id,transaction_hash,log_index)
);
CREATE TABLE IF NOT EXISTS funding_intent_receipt_migration_conflicts (
 network TEXT NOT NULL,
 asset_id TEXT NOT NULL,
 transaction_hash TEXT NOT NULL,
 log_index INTEGER NOT NULL CHECK(log_index >= 0),
 intent_id TEXT NOT NULL,
 wallet_id TEXT NOT NULL,
 atomic_amount TEXT NOT NULL,
 recipient TEXT NOT NULL,
 block_hash TEXT NOT NULL,
 check_id TEXT NOT NULL,
 delivery_id TEXT NOT NULL,
 created_at TEXT NOT NULL,
 PRIMARY KEY(network,asset_id,transaction_hash,log_index,intent_id)
);
INSERT OR IGNORE INTO funding_intent_receipt_migration_conflicts(
 network,asset_id,transaction_hash,log_index,intent_id,wallet_id,atomic_amount,recipient,block_hash,check_id,delivery_id,created_at
)
 SELECT b.network,b.asset_id,LOWER(b.transaction_hash),b.log_index,b.intent_id,b.wallet_id,b.atomic_amount,b.recipient,b.block_hash,b.check_id,b.delivery_id,b.created_at
 FROM funding_intent_receipt_bindings b
 WHERE EXISTS (
  SELECT 1 FROM funding_intent_receipt_bindings other
  WHERE other.network=b.network AND other.asset_id=b.asset_id
   AND LOWER(other.transaction_hash)=LOWER(b.transaction_hash)
   AND other.log_index=b.log_index AND other.intent_id<>b.intent_id
 );
CREATE TRIGGER IF NOT EXISTS funding_intent_receipt_claim_conflict_guard
 BEFORE INSERT ON funding_intent_receipt_log_claims
 WHEN EXISTS (
  SELECT 1 FROM funding_intent_receipt_migration_conflicts c
  WHERE c.network=NEW.network AND c.asset_id=NEW.asset_id
   AND c.transaction_hash=NEW.transaction_hash AND c.log_index=NEW.log_index
 )
 BEGIN
  SELECT RAISE(ABORT,'conflicting historical funding receipt bindings require operator reconciliation');
 END;
INSERT OR IGNORE INTO funding_intent_receipt_log_claims(network,asset_id,transaction_hash,log_index,intent_id,created_at)
 SELECT network,asset_id,LOWER(transaction_hash),log_index,intent_id,created_at FROM funding_intent_receipt_bindings;
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
    -- Legacy column name; this is an opaque per-generation reference, not a
    -- wallet row. Session provisioning must not mint another signing key.
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

-- Operator-initiated session rotations reserve one target generation per stable
-- request key. The result snapshot makes response-loss retries idempotent.
CREATE TABLE IF NOT EXISTS mandate_session_rotations (
    mandate_id TEXT NOT NULL,
    operator_identity TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    target_generation INTEGER NOT NULL CHECK (target_generation > 0),
    result_json BLOB NOT NULL DEFAULT X'',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY(mandate_id,operator_identity,idempotency_key),
    UNIQUE(mandate_id,target_generation)
);
