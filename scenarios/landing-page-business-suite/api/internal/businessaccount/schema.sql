-- LPBS commercial-account context and membership.
-- User IDs are soft references because authentication owns the users table;
-- this domain owns account membership and selection authorization.
CREATE TABLE IF NOT EXISTS business_accounts (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    billing_email TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS business_account_members (
    business_account_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (business_account_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_business_account_members_user
    ON business_account_members(user_id, business_account_id);
