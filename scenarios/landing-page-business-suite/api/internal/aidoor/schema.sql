CREATE TABLE IF NOT EXISTS ai_access_keys (
 id TEXT PRIMARY KEY,
 token_hash TEXT NOT NULL UNIQUE,
 parent_id TEXT NOT NULL DEFAULT '',
 kind TEXT NOT NULL,
 business_account_id TEXT NOT NULL,
 actor_identity TEXT NOT NULL,
 app_bundle_key TEXT NOT NULL,
 harness TEXT NOT NULL DEFAULT '',
 model TEXT NOT NULL DEFAULT '',
 max_credits BIGINT NOT NULL,
 expires_at BIGINT NOT NULL,
 revoked BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE TABLE IF NOT EXISTS ai_key_charges (
 operation_id TEXT NOT NULL,
 key_id TEXT NOT NULL,
 reserved_credits BIGINT NOT NULL,
 actual_credits BIGINT,
 PRIMARY KEY(operation_id,key_id)
);
