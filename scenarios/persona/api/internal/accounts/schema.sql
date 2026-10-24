CREATE TABLE IF NOT EXISTS persona_account_links (
  id TEXT PRIMARY KEY NOT NULL,
  persona_id TEXT NOT NULL REFERENCES personas(id),
  site TEXT NOT NULL,
  login_seam TEXT NOT NULL,
  recovery_path TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS persona_addresses (
  id TEXT PRIMARY KEY NOT NULL,
  persona_id TEXT NOT NULL REFERENCES personas(id),
  label TEXT NOT NULL,
  line1 TEXT NOT NULL,
  line2 TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL,
  region TEXT NOT NULL,
  postal_code TEXT NOT NULL,
  country TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS persona_obligations (
  id TEXT PRIMARY KEY NOT NULL,
  persona_id TEXT NOT NULL REFERENCES personas(id),
  account_link_id TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL,
  renewal_at TEXT NOT NULL,
  cancel_path TEXT NOT NULL,
  cancelled INTEGER NOT NULL CHECK (cancelled IN (0, 1)),
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS persona_account_links_persona_idx ON persona_account_links(persona_id, created_at DESC);
CREATE INDEX IF NOT EXISTS persona_addresses_persona_idx ON persona_addresses(persona_id, created_at DESC);
CREATE INDEX IF NOT EXISTS persona_obligations_persona_idx ON persona_obligations(persona_id, renewal_at);
