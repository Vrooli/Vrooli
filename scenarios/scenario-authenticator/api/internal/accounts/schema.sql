-- Accounts + realms tables — owned by internal/accounts/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry. Times are RFC3339Nano strings (matching the
-- notes domain and the time.Time round-trip in sqlite.go). Forward-only
-- declarative: CREATE TABLE IF NOT EXISTS so re-runs are no-ops; additive
-- columns land as a one-shot migration, never a recreate (storage-steer §5).

-- realms is the multi-tenant boundary. At P0 a single "default" realm is
-- seeded; tokens are aud-scoped to the realm even with one tenant.
CREATE TABLE IF NOT EXISTS realms (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL,
  audience   TEXT NOT NULL,
  created_at TEXT NOT NULL
);

-- Seed the default realm. INSERT OR IGNORE keeps boot idempotent.
INSERT OR IGNORE INTO realms (id, name, audience, created_at)
VALUES ('default', 'Default', 'scenario-authenticator:default', '1970-01-01T00:00:00Z');

-- accounts are realm-scoped: email is unique within a realm, not globally.
-- password_hash is an argon2id PHC string (never plaintext). roles is a JSON
-- array. failed_login_attempts/locked_until back account lockout (the old
-- scenario declared these columns but never enforced them).
CREATE TABLE IF NOT EXISTS accounts (
  id                    TEXT PRIMARY KEY,
  realm_id              TEXT NOT NULL,
  email                 TEXT NOT NULL,
  username              TEXT NOT NULL DEFAULT '',
  password_hash         TEXT NOT NULL,
  roles                 TEXT NOT NULL DEFAULT '["user"]',
  email_verified        INTEGER NOT NULL DEFAULT 0,
  failed_login_attempts INTEGER NOT NULL DEFAULT 0,
  locked_until          TEXT NOT NULL DEFAULT '',
  created_at            TEXT NOT NULL,
  updated_at            TEXT NOT NULL,
  last_login            TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (realm_id) REFERENCES realms(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_realm_email ON accounts(realm_id, email);

-- A local principal may have many account bindings. Resolution is deliberately
-- single-valued: exactly one is_default row must exist or exchange refuses.
CREATE TABLE IF NOT EXISTS machine_bindings (
  id               TEXT PRIMARY KEY,
  machine_id       TEXT NOT NULL,
  local_principal  TEXT NOT NULL,
  account_id       TEXT NOT NULL,
  realm_id         TEXT NOT NULL,
  is_default       INTEGER NOT NULL DEFAULT 0,
  linked_at        TEXT NOT NULL,
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (realm_id) REFERENCES realms(id)
);

CREATE INDEX IF NOT EXISTS idx_machine_bindings_lookup
  ON machine_bindings(machine_id, local_principal, is_default, linked_at);
