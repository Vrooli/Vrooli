-- Pairing domain tables — owned by internal/pairing/. Applied via
-- database.EnsureSchemas at boot (modules.AllSchemas). One-touch bootstrap +
-- mutual-auth enrollment (OT-P0-002). Times are RFC3339Nano strings matching
-- the wire format. CREATE TABLE IF NOT EXISTS so re-runs are no-ops (migrate,
-- never recreate); Postgres-compatible column types for forward scale.

-- pairing_codes — single-use, short-TTL bootstrap tokens. Only the HASH of the
-- code is stored (the plaintext is shown once at issue); a live code can enrol a
-- rogue node, so it is the one genuinely-secret artifact here. redeemed_at burns
-- it on use (single-use), independent of expires_at (TTL).
CREATE TABLE IF NOT EXISTS pairing_codes (
  id               TEXT PRIMARY KEY,
  code_hash        TEXT NOT NULL UNIQUE,
  name             TEXT NOT NULL DEFAULT '',
  scopes           TEXT NOT NULL DEFAULT '[]',
  created_at       TEXT NOT NULL,
  expires_at       TEXT NOT NULL,
  redeemed_at      TEXT NOT NULL DEFAULT '',
  redeemed_node_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_pairing_codes_expires_at ON pairing_codes(expires_at);

-- node_credentials — the node's Ed25519 PUBLIC key (standard base64). This is
-- NOT secret material: with asymmetric mutual auth the node keeps the private
-- key and it never leaves the node, so there is nothing to hash. The control
-- plane stores the public key in the clear precisely because it must stay
-- verifiable. revoked_at severs auth instantly (nodeauth treats a revoked
-- credential as absent).
CREATE TABLE IF NOT EXISTS node_credentials (
  node_id     TEXT PRIMARY KEY,
  public_key  TEXT NOT NULL,
  created_at  TEXT NOT NULL,
  revoked_at  TEXT NOT NULL DEFAULT ''
);

-- pairing_requests — the request/approve fallback when there is no pre-shared
-- code. A node asks to join (status pending) and the owner approves/rejects.
CREATE TABLE IF NOT EXISTS pairing_requests (
  id           TEXT PRIMARY KEY,
  public_key   TEXT NOT NULL,
  name         TEXT NOT NULL DEFAULT '',
  os           TEXT NOT NULL DEFAULT '',
  arch         TEXT NOT NULL DEFAULT '',
  endpoint     TEXT NOT NULL DEFAULT '',
  capabilities TEXT NOT NULL DEFAULT '[]',
  status       TEXT NOT NULL DEFAULT 'pending',
  node_id      TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL,
  decided_at   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_pairing_requests_status ON pairing_requests(status, created_at DESC);
