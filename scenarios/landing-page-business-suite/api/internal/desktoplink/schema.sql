-- Durable LPBS-to-desktop account links and one-use authorization transactions.
-- Identity IDs are soft references: LPBS owns its user/account records and the
-- local authenticator owns local principals. This domain owns only the link.
CREATE TABLE IF NOT EXISTS desktop_link_authorizations (
    id TEXT PRIMARY KEY,
    code_hash TEXT NOT NULL UNIQUE,
    lpbs_user_id TEXT NOT NULL,
    business_account_id TEXT NOT NULL,
    installation_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    audience TEXT NOT NULL,
    scopes_json TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_desktop_link_auth_expiry ON desktop_link_authorizations(expires_at, used_at);

CREATE TABLE IF NOT EXISTS desktop_account_links (
    id TEXT PRIMARY KEY,
    lpbs_user_id TEXT NOT NULL,
    business_account_id TEXT NOT NULL,
    local_provider TEXT NOT NULL,
    local_principal TEXT NOT NULL,
    installation_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    audience TEXT NOT NULL,
    scopes_json TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP,
    revoked_by TEXT
);
CREATE INDEX IF NOT EXISTS idx_desktop_account_links_lpbs ON desktop_account_links(lpbs_user_id, revoked_at);
CREATE INDEX IF NOT EXISTS idx_desktop_account_links_local ON desktop_account_links(local_principal, installation_id, revoked_at);

CREATE TABLE IF NOT EXISTS desktop_link_audit (
    id TEXT PRIMARY KEY,
    link_id TEXT,
    authorization_id TEXT,
    event TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    installation_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    details_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_desktop_link_audit_link ON desktop_link_audit(link_id, created_at);
