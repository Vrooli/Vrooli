-- Credit consumption, API keys, and user authentication.
CREATE TABLE IF NOT EXISTS subscription_tier_limits (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), tier_id VARCHAR(50) NOT NULL, limit_type VARCHAR(20) NOT NULL, limit_key VARCHAR(100) NOT NULL, limit_value BIGINT NOT NULL, cost_multiplier BIGINT DEFAULT 1000000, app_bundle_key VARCHAR(100), reset_period VARCHAR(20) DEFAULT 'monthly', created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW(), UNIQUE(tier_id, limit_type, limit_key, app_bundle_key));
CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_tier ON subscription_tier_limits(tier_id); CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_type ON subscription_tier_limits(limit_type); CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_app ON subscription_tier_limits(app_bundle_key);
CREATE TABLE IF NOT EXISTS usage_records (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_identity VARCHAR(255) NOT NULL, billing_period VARCHAR(20) NOT NULL, limit_key VARCHAR(100) NOT NULL, usage_amount BIGINT NOT NULL DEFAULT 0, app_bundle_key VARCHAR(100), operation_id UUID, last_operation_at TIMESTAMP, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW(), UNIQUE(user_identity, billing_period, limit_key, app_bundle_key));
CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_identity_period_key_app ON usage_records(user_identity, billing_period, limit_key, app_bundle_key); CREATE INDEX IF NOT EXISTS idx_usage_records_user_period ON usage_records(user_identity, billing_period); CREATE INDEX IF NOT EXISTS idx_usage_records_limit_key ON usage_records(limit_key); CREATE INDEX IF NOT EXISTS idx_usage_records_app ON usage_records(app_bundle_key); CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_operation_id ON usage_records(operation_id) WHERE operation_id IS NOT NULL;
CREATE TABLE IF NOT EXISTS api_keys (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), provider VARCHAR(50) NOT NULL UNIQUE, encrypted_key TEXT NOT NULL, key_hint VARCHAR(20), is_active BOOLEAN DEFAULT true, last_verified_at TIMESTAMP, encryption_state VARCHAR(16) NOT NULL DEFAULT 'unknown', created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_api_keys_provider ON api_keys(provider); CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);
CREATE TABLE IF NOT EXISTS credit_reservations (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_identity VARCHAR(255) NOT NULL, billing_period VARCHAR(20) NOT NULL, limit_key VARCHAR(100) NOT NULL, reserved_amount BIGINT NOT NULL, status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','finalized','released','expired')), created_at TIMESTAMP DEFAULT NOW(), finalized_at TIMESTAMP, expires_at TIMESTAMP NOT NULL);
CREATE INDEX IF NOT EXISTS idx_credit_reservations_user ON credit_reservations(user_identity, status); CREATE INDEX IF NOT EXISTS idx_credit_reservations_expires ON credit_reservations(expires_at) WHERE status = 'pending';
CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, email_verified BOOLEAN DEFAULT FALSE, stripe_customer_id VARCHAR(255), has_used_intro BOOLEAN DEFAULT FALSE, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW(), last_login_at TIMESTAMP, webauthn_user_handle BYTEA UNIQUE);
CREATE TABLE IF NOT EXISTS user_passkeys (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, credential_id BYTEA NOT NULL UNIQUE, public_key BYTEA NOT NULL, sign_count BIGINT NOT NULL DEFAULT 0, aaguid BYTEA, nickname VARCHAR(64) NOT NULL DEFAULT 'Passkey', backup_state VARCHAR(32), rp_id VARCHAR(255) NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT NOW(), last_used_at TIMESTAMP, revoked_at TIMESTAMP);
CREATE INDEX IF NOT EXISTS idx_user_passkeys_user ON user_passkeys(user_id) WHERE revoked_at IS NULL;
CREATE TABLE IF NOT EXISTS webauthn_challenges (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), challenge_hash CHAR(64) NOT NULL, purpose VARCHAR(40) NOT NULL, subject TEXT NOT NULL, session_data JSONB NOT NULL, binding_hash CHAR(64), sign_in_context JSONB NOT NULL DEFAULT '{}', expires_at TIMESTAMP NOT NULL, consumed_at TIMESTAMP, created_at TIMESTAMP NOT NULL DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_expiry ON webauthn_challenges(expires_at, consumed_at);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email); CREATE INDEX IF NOT EXISTS idx_users_stripe_customer ON users(stripe_customer_id); CREATE INDEX IF NOT EXISTS idx_users_has_used_intro ON users(has_used_intro);
CREATE TABLE IF NOT EXISTS auth_tokens (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, token_hash VARCHAR(255) NOT NULL, token_type VARCHAR(50) NOT NULL, expires_at TIMESTAMP NOT NULL, used_at TIMESTAMP, created_at TIMESTAMP DEFAULT NOW(), ip_address INET, user_agent TEXT, email VARCHAR(255), code_hash VARCHAR(255), request_context JSONB, browser_binding_hash VARCHAR(255), delivery_status VARCHAR(20), delivery_error TEXT, provider VARCHAR(20), provider_message_id VARCHAR(255), provider_status VARCHAR(20), provider_status_at TIMESTAMP, provider_reason_class VARCHAR(40));
CREATE INDEX IF NOT EXISTS idx_auth_tokens_hash ON auth_tokens(token_hash); CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_expires ON auth_tokens(user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_expires ON auth_tokens(expires_at); CREATE INDEX IF NOT EXISTS idx_auth_tokens_delivery ON auth_tokens(delivery_status, created_at);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_provider_message ON auth_tokens(provider_message_id);
CREATE TABLE IF NOT EXISTS auth_email_events (sg_event_id VARCHAR(128) PRIMARY KEY, sg_message_id VARCHAR(255), sign_in_request_id UUID, event VARCHAR(32) NOT NULL, event_name VARCHAR(30) NOT NULL, reason_class VARCHAR(40), occurred_at TIMESTAMP NOT NULL, event_timestamp TIMESTAMP NOT NULL, email VARCHAR(255), provider_status VARCHAR(30) NOT NULL, raw_custom_args JSONB NOT NULL DEFAULT '{}', received_at TIMESTAMP NOT NULL DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_auth_email_events_received ON auth_email_events(received_at); CREATE INDEX IF NOT EXISTS idx_auth_email_events_status ON auth_email_events(provider_status, received_at);
CREATE INDEX IF NOT EXISTS idx_auth_email_events_request ON auth_email_events(sign_in_request_id, occurred_at);
CREATE TABLE IF NOT EXISTS auth_alert_log (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), alert_type VARCHAR(80) NOT NULL, window_start TIMESTAMP NOT NULL DEFAULT NOW(), details JSONB NOT NULL DEFAULT '{}', dispatch_status VARCHAR(20) NOT NULL DEFAULT 'pending', dispatch_error TEXT, created_at TIMESTAMP NOT NULL DEFAULT NOW(), UNIQUE(alert_type, window_start));
-- Durable authentication throttles shared by every API replica and preserved across restarts.
CREATE TABLE IF NOT EXISTS auth_rate_events (id BIGSERIAL PRIMARY KEY, bucket TEXT NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_auth_rate_events_bucket ON auth_rate_events(bucket, created_at); CREATE INDEX IF NOT EXISTS idx_auth_rate_events_created ON auth_rate_events(created_at);
CREATE TABLE IF NOT EXISTS user_sessions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, refresh_token_hash VARCHAR(255) NOT NULL, refresh_token_family_id UUID NOT NULL DEFAULT gen_random_uuid(), expires_at TIMESTAMP NOT NULL, created_at TIMESTAMP DEFAULT NOW(), last_used_at TIMESTAMP DEFAULT NOW(), ip_address INET, user_agent TEXT, device_info JSONB DEFAULT '{}', revoked BOOLEAN DEFAULT FALSE, authenticated_at TIMESTAMP NOT NULL DEFAULT NOW(), absolute_expires_at TIMESTAMP NOT NULL DEFAULT (NOW() + INTERVAL '90 days'), auth_method VARCHAR(30) NOT NULL DEFAULT 'email_code', revoked_at TIMESTAMP, revoked_reason VARCHAR(40));
CREATE TABLE IF NOT EXISTS native_auth_grants (code_hash CHAR(64) PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, code_challenge TEXT NOT NULL, redirect_uri TEXT NOT NULL, browser_binding_hash VARCHAR(255), sign_in_ip INET, sign_in_user_agent TEXT, authenticated_at TIMESTAMP NOT NULL, session_id UUID, expires_at TIMESTAMP NOT NULL, used_at TIMESTAMP, created_at TIMESTAMP NOT NULL DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_native_auth_grants_expiry ON native_auth_grants(expires_at, used_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id); CREATE INDEX IF NOT EXISTS idx_user_sessions_hash ON user_sessions(refresh_token_hash); CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(user_id, revoked, expires_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_absolute ON user_sessions(absolute_expires_at);
CREATE TABLE IF NOT EXISTS refresh_token_history (refresh_token_hash VARCHAR(255) PRIMARY KEY, session_id UUID NOT NULL REFERENCES user_sessions(id) ON DELETE CASCADE, family_id UUID NOT NULL, retired_at TIMESTAMP NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS usage_events (id BIGSERIAL PRIMARY KEY, operation_id TEXT UNIQUE, user_identity VARCHAR(255) NOT NULL, app_bundle_key VARCHAR(100) NOT NULL DEFAULT 'unattributed', model VARCHAR(255) NOT NULL DEFAULT 'unknown', credits BIGINT NOT NULL, cost_micros BIGINT NOT NULL DEFAULT 0, prompt_tokens BIGINT NOT NULL DEFAULT 0, completion_tokens BIGINT NOT NULL DEFAULT 0, provider VARCHAR(100) NOT NULL DEFAULT 'local', created_at TIMESTAMP NOT NULL DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_usage_events_created ON usage_events(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_events_app_model ON usage_events(app_bundle_key, model, created_at);

-- Business-account metering is a separate audience from a person's account.
-- Actor identity remains on each reservation and event for audit attribution.
CREATE TABLE IF NOT EXISTS business_account_usage_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_account_id TEXT NOT NULL,
  actor_identity VARCHAR(255) NOT NULL,
  billing_period VARCHAR(20) NOT NULL,
  limit_key VARCHAR(100) NOT NULL,
  usage_amount BIGINT NOT NULL DEFAULT 0,
  app_bundle_key VARCHAR(100) NOT NULL DEFAULT 'unattributed',
  last_operation_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(business_account_id, billing_period, limit_key, app_bundle_key)
);
CREATE INDEX IF NOT EXISTS idx_business_account_usage_period ON business_account_usage_records(business_account_id, billing_period, limit_key);
CREATE TABLE IF NOT EXISTS business_account_credit_reservations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_account_id TEXT NOT NULL,
  actor_identity VARCHAR(255) NOT NULL,
  billing_period VARCHAR(20) NOT NULL,
  limit_key VARCHAR(100) NOT NULL,
  reserved_amount BIGINT NOT NULL,
  livemode BOOLEAN,
  actual_amount BIGINT,
  charged_amount BIGINT NOT NULL DEFAULT 0,
  overrun_amount BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  settled_at TIMESTAMP,
  app_bundle_key VARCHAR(100) NOT NULL DEFAULT 'unattributed',
  model VARCHAR(255) NOT NULL DEFAULT 'unknown',
  cost_micros BIGINT NOT NULL DEFAULT 0,
  prompt_tokens BIGINT NOT NULL DEFAULT 0,
  completion_tokens BIGINT NOT NULL DEFAULT 0,
  provider VARCHAR(100) NOT NULL DEFAULT 'openrouter'
);
ALTER TABLE business_account_credit_reservations ADD COLUMN IF NOT EXISTS charged_amount BIGINT NOT NULL DEFAULT 0;
ALTER TABLE business_account_credit_reservations ADD COLUMN IF NOT EXISTS overrun_amount BIGINT NOT NULL DEFAULT 0;
-- Null means the funding mode was not recorded. Never infer it from an
-- account's current lots; active unknown-mode holds fail closed.
ALTER TABLE business_account_credit_reservations ADD COLUMN IF NOT EXISTS livemode BOOLEAN;
CREATE TABLE IF NOT EXISTS credit_funding_lots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_account_id TEXT NOT NULL,
  source_kind VARCHAR(40) NOT NULL,
  provider VARCHAR(40) NOT NULL DEFAULT 'unknown',
  provider_ref TEXT,
  provider_ref2 TEXT,
  external_event_id TEXT,
  livemode BOOLEAN,
  settlement_state VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (settlement_state IN ('pending','settled','failed','reversed')),
  spendable_after TIMESTAMP,
  reversal_window_ends TIMESTAMP,
  reversal_risk VARCHAR(32),
  paid_amount_minor BIGINT,
  paid_currency CHAR(3),
  cost_basis_usd_micros BIGINT,
  basis_rate_ref TEXT,
  granted BIGINT NOT NULL CHECK (granted >= 0),
  remaining BIGINT NOT NULL CHECK (remaining >= 0 AND remaining <= granted),
  expires_at TIMESTAMP,
  status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open','frozen','exhausted','expired','reversed')),
  funding_fingerprint TEXT,
  policy_id TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE (business_account_id, provider, provider_ref)
);
CREATE INDEX IF NOT EXISTS idx_credit_funding_lots_spendable ON credit_funding_lots(business_account_id, status, settlement_state, livemode, spendable_after, expires_at);
CREATE TABLE IF NOT EXISTS credit_lot_allocations (
  id TEXT PRIMARY KEY,
  lot_id UUID NOT NULL REFERENCES credit_funding_lots(id),
  operation_id TEXT NOT NULL,
  credits BIGINT NOT NULL CHECK (credits > 0),
  cost_micros BIGINT NOT NULL DEFAULT 0 CHECK (cost_micros >= 0),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE (lot_id, operation_id)
);
CREATE INDEX IF NOT EXISTS idx_credit_lot_allocations_operation ON credit_lot_allocations(operation_id);
CREATE TABLE IF NOT EXISTS funding_losses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lot_id UUID NOT NULL REFERENCES credit_funding_lots(id),
  kind VARCHAR(40) NOT NULL,
  consumed_cost_micros BIGINT NOT NULL DEFAULT 0,
  fee_minor BIGINT NOT NULL DEFAULT 0,
  reversed_revenue_minor BIGINT NOT NULL DEFAULT 0,
  refunded_revenue_minor BIGINT NOT NULL DEFAULT 0,
  disputed_revenue_minor BIGINT NOT NULL DEFAULT 0,
  recorded_to_account TEXT NOT NULL,
  event_ref TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE (lot_id, event_ref, kind)
);
CREATE INDEX IF NOT EXISTS idx_funding_losses_account ON funding_losses(recorded_to_account, created_at);
-- Provider lifecycle is distinct from lot exhaustion. Replays and late delivery
-- cannot reopen a finalized dispute or increase an already-applied refund.
CREATE TABLE IF NOT EXISTS funding_payment_states (
  lot_id UUID PRIMARY KEY REFERENCES credit_funding_lots(id),
  fraud_warning BOOLEAN NOT NULL DEFAULT FALSE,
  restricted_at TIMESTAMP,
  refunded_minor BIGINT NOT NULL DEFAULT 0 CHECK (refunded_minor >= 0)
);
CREATE TABLE IF NOT EXISTS funding_disputes (
  lot_id UUID NOT NULL REFERENCES credit_funding_lots(id),
  dispute_id TEXT NOT NULL,
  status TEXT NOT NULL,
  amount_minor BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY(lot_id,dispute_id)
);
-- Immutable Stripe balance transactions own actual withdrawn/reinstated
-- principal and fees, including partial returns whose dispute stays lost.
CREATE TABLE IF NOT EXISTS funding_dispute_balance_transactions (
  lot_id UUID NOT NULL,
  dispute_id TEXT NOT NULL,
  transaction_id TEXT NOT NULL,
  amount_minor BIGINT NOT NULL,
  fee_minor BIGINT NOT NULL,
  currency CHAR(3) NOT NULL,
  PRIMARY KEY(lot_id,dispute_id,transaction_id),
  FOREIGN KEY(lot_id,dispute_id) REFERENCES funding_disputes(lot_id,dispute_id)
);
CREATE TABLE IF NOT EXISTS funding_provider_events (
  event_id TEXT NOT NULL,
  livemode BOOLEAN NOT NULL,
  payload JSONB NOT NULL,
  PRIMARY KEY(event_id,livemode)
);
CREATE TABLE IF NOT EXISTS funding_purchase_blocks (
  business_account_id TEXT PRIMARY KEY,
  reason TEXT NOT NULL,
  event_ref TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
-- No production trust values are seeded. Operator-owned policy is data;
-- journey fixtures insert isolated proposals without approving a trust bar.
CREATE TABLE IF NOT EXISTS funding_source_policies (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  source_kind TEXT NOT NULL,
  young_account_seconds BIGINT NOT NULL CHECK (young_account_seconds >= 0),
  young_hold_seconds BIGINT NOT NULL CHECK (young_hold_seconds >= 0),
  young_cap_credits BIGINT NOT NULL CHECK (young_cap_credits >= 0),
  mature_cap_credits BIGINT NOT NULL CHECK (mature_cap_credits >= 0),
  UNIQUE(provider,source_kind)
);
CREATE INDEX IF NOT EXISTS idx_business_account_reservations_active ON business_account_credit_reservations(business_account_id, billing_period, limit_key, status);
CREATE INDEX IF NOT EXISTS idx_business_account_reservations_expiry ON business_account_credit_reservations(status, expires_at);
CREATE TABLE IF NOT EXISTS business_account_usage_events (
  id BIGSERIAL PRIMARY KEY,
  operation_id TEXT NOT NULL UNIQUE,
  business_account_id TEXT NOT NULL,
  actor_identity VARCHAR(255) NOT NULL,
  app_bundle_key VARCHAR(100) NOT NULL DEFAULT 'unattributed',
  model VARCHAR(255) NOT NULL DEFAULT 'unknown',
  credits BIGINT NOT NULL,
  cost_micros BIGINT NOT NULL DEFAULT 0,
  prompt_tokens BIGINT NOT NULL DEFAULT 0,
  completion_tokens BIGINT NOT NULL DEFAULT 0,
  provider VARCHAR(100) NOT NULL DEFAULT 'openrouter',
  feature_id TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
ALTER TABLE business_account_usage_events ADD COLUMN IF NOT EXISTS feature_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_business_account_usage_events_account ON business_account_usage_events(business_account_id, created_at);
-- The outbox commits with the lot mutation, independently of ledger uptime.
-- Separate mode-specific sinks prevent test reversals reaching a live book.
CREATE TABLE IF NOT EXISTS funding_ledger_sinks (
  livemode BOOLEAN PRIMARY KEY,
  endpoint TEXT NOT NULL,
  adapter_id TEXT NOT NULL,
  book_id TEXT NOT NULL,
  loss_account_id TEXT NOT NULL,
  refund_account_id TEXT NOT NULL,
  fee_account_id TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS funding_ledger_outbox (
  id BIGSERIAL PRIMARY KEY,
  lot_id UUID NOT NULL REFERENCES credit_funding_lots(id),
  category TEXT NOT NULL,
  amount_minor BIGINT NOT NULL CHECK(amount_minor<>0),
  currency TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  delivered_at TIMESTAMP
);
