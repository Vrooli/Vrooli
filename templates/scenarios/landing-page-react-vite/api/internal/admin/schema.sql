-- Admin domain schema.
--
-- Admin operator accounts authenticated with bcrypt password hashes. Sessions
-- are stateless signed cookies (see internal/admin), so no session table is
-- needed. Forward-only declarative DDL.
CREATE TABLE IF NOT EXISTS admin_users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_admin_users_email ON admin_users(email);
