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
  correlation_id   TEXT NOT NULL DEFAULT '',
  created_at       TEXT NOT NULL,
  expires_at       TEXT NOT NULL,
  claimed_at       TEXT NOT NULL DEFAULT '',
  redeemed_at      TEXT NOT NULL DEFAULT '',
  redeemed_node_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_pairing_codes_expires_at ON pairing_codes(expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pairing_codes_correlation
  ON pairing_codes(correlation_id) WHERE correlation_id <> '';

-- A durable saga record is written before a correlated redemption can create a
-- Registry Node. It contains only public key/facts and enables restart-safe
-- reconciliation of the otherwise cross-domain Registry/credential/code work.
CREATE TABLE IF NOT EXISTS pairing_enrollment_sagas (
  correlation_id TEXT PRIMARY KEY,
  code_id TEXT NOT NULL,
  public_key TEXT NOT NULL,
  facts TEXT NOT NULL DEFAULT '{}',
  state TEXT NOT NULL DEFAULT 'prepared',
  node_id TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_pairing_enrollment_sagas_state
  ON pairing_enrollment_sagas(state, created_at);

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

CREATE TABLE IF NOT EXISTS node_encryption_keys (
  node_id     TEXT PRIMARY KEY,
  public_key  TEXT NOT NULL,
  algorithm   TEXT NOT NULL DEFAULT 'x25519',
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
