-- Admin identity and remote-session administration.
CREATE TABLE IF NOT EXISTS admin_users (id SERIAL PRIMARY KEY, email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, created_at TIMESTAMP DEFAULT NOW(), last_login TIMESTAMP);
CREATE INDEX IF NOT EXISTS idx_admin_users_email ON admin_users(email);
CREATE TABLE IF NOT EXISTS remote_profiles (id SERIAL PRIMARY KEY, tag VARCHAR(100) UNIQUE NOT NULL, label TEXT, api_base TEXT NOT NULL, connector_id VARCHAR(64) UNIQUE, remote_session_id TEXT, status VARCHAR(20) NOT NULL DEFAULT 'unknown' CHECK (status IN ('unknown','active','expired','error')), encrypted_session TEXT, session_expires_at TIMESTAMP, remote_session_last_synced_at TIMESTAMP, last_login_at TIMESTAMP, last_used_at TIMESTAMP, created_by INTEGER REFERENCES admin_users(id) ON DELETE SET NULL, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW());
CREATE UNIQUE INDEX IF NOT EXISTS idx_remote_profiles_connector_id ON remote_profiles(connector_id);
CREATE INDEX IF NOT EXISTS idx_remote_profiles_tag ON remote_profiles(tag);
CREATE INDEX IF NOT EXISTS idx_remote_profiles_status ON remote_profiles(status);
CREATE TABLE IF NOT EXISTS admin_sessions (id TEXT PRIMARY KEY, admin_email TEXT NOT NULL, created_at TIMESTAMP DEFAULT NOW(), last_activity TIMESTAMP DEFAULT NOW(), expires_at TIMESTAMP NOT NULL, ip_address TEXT, user_agent TEXT);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_email ON admin_sessions(admin_email);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires ON admin_sessions(expires_at);
