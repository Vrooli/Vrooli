-- Admin identity and remote-session administration.
CREATE TABLE IF NOT EXISTS admin_users (id SERIAL PRIMARY KEY, email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, created_at TIMESTAMP DEFAULT NOW(), last_login TIMESTAMP, totp_secret_encrypted TEXT, totp_pending_secret_encrypted TEXT, totp_enabled_at TIMESTAMP, totp_last_step BIGINT, recovery_code_hashes JSONB, webauthn_user_handle BYTEA UNIQUE);
CREATE TABLE IF NOT EXISTS admin_passkeys (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), admin_id INTEGER NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE, credential_id BYTEA NOT NULL UNIQUE, public_key BYTEA NOT NULL, sign_count BIGINT NOT NULL DEFAULT 0, aaguid BYTEA, nickname VARCHAR(64) NOT NULL DEFAULT 'Passkey', backup_state VARCHAR(32), rp_id VARCHAR(255) NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT NOW(), last_used_at TIMESTAMP, revoked_at TIMESTAMP);
CREATE INDEX IF NOT EXISTS idx_admin_passkeys_admin ON admin_passkeys(admin_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admin_users_email ON admin_users(email);
CREATE TABLE IF NOT EXISTS remote_profiles (id SERIAL PRIMARY KEY, tag VARCHAR(100) UNIQUE NOT NULL, label TEXT, api_base TEXT NOT NULL, auth_mode VARCHAR(16) NOT NULL DEFAULT 'session' CHECK (auth_mode IN ('session','service')), encrypted_remote_service_secret TEXT, connector_id VARCHAR(64) UNIQUE, remote_session_id TEXT, status VARCHAR(20) NOT NULL DEFAULT 'unknown' CHECK (status IN ('unknown','active','expired','error')), encrypted_session TEXT, encryption_state VARCHAR(16) NOT NULL DEFAULT 'unknown', session_expires_at TIMESTAMP, remote_session_last_synced_at TIMESTAMP, last_login_at TIMESTAMP, last_used_at TIMESTAMP, created_by INTEGER REFERENCES admin_users(id) ON DELETE SET NULL, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW());
CREATE UNIQUE INDEX IF NOT EXISTS idx_remote_profiles_connector_id ON remote_profiles(connector_id);
CREATE INDEX IF NOT EXISTS idx_remote_profiles_tag ON remote_profiles(tag);
CREATE INDEX IF NOT EXISTS idx_remote_profiles_status ON remote_profiles(status);
CREATE TABLE IF NOT EXISTS admin_sessions (id TEXT PRIMARY KEY, admin_email TEXT NOT NULL, created_at TIMESTAMP DEFAULT NOW(), last_activity TIMESTAMP DEFAULT NOW(), expires_at TIMESTAMP NOT NULL, ip_address TEXT, user_agent TEXT, assurance VARCHAR(20) NOT NULL DEFAULT 'full', reauthenticated_at TIMESTAMP);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_email ON admin_sessions(admin_email);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires ON admin_sessions(expires_at);
CREATE TABLE IF NOT EXISTS admin_security_events (id BIGSERIAL PRIMARY KEY, admin_email TEXT NOT NULL, session_id_hash CHAR(64), event_type VARCHAR(64) NOT NULL, ip_address TEXT, user_agent TEXT, detail JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMP NOT NULL DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_admin_security_events_email_time ON admin_security_events(admin_email, created_at DESC);
CREATE TABLE IF NOT EXISTS credential_mint_witness (
  logical_id TEXT NOT NULL,
  field TEXT NOT NULL,
  minted_at TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (logical_id, field)
);
CREATE TABLE IF NOT EXISTS reader_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), label TEXT NOT NULL,
  scope TEXT NOT NULL, token_sha256 CHAR(64) NOT NULL UNIQUE, prefix VARCHAR(32) NOT NULL,
  source TEXT NOT NULL DEFAULT 'operator', created_by INTEGER REFERENCES admin_users(id) ON DELETE SET NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(), last_used_at TIMESTAMP, revoked_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reader_tokens_scope_active ON reader_tokens(scope, revoked_at);
