-- Devices domain — owned by internal/devices/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot (through modules.AllSchemas).
-- Times are RFC3339Nano strings matching the wire format and the time.Time
-- round-trip in sqlite.go. Forward-only: CREATE TABLE IF NOT EXISTS so re-runs
-- are no-ops; column additions land as `ALTER TABLE ... ADD COLUMN` (never
-- recreate — Vrooli storage rule).

-- hub_owner is the single-owner claim: exactly one row (id = 1) records which
-- authenticator identity owns this hub. The CHECK pins it to a singleton; the
-- claim is first-owner-wins via a conditional insert (INSERT ... ON CONFLICT(id)
-- DO NOTHING) so two concurrent SetupOwnerDevice calls can never both win.
-- Owner-authed RPCs verify the caller matches owner_id (single-owner model).
-- The owner is established by the first SetupOwnerDevice call; ownership is an
-- explicit hub fact, never derived from device rows.
CREATE TABLE IF NOT EXISTS hub_owner (
  id         INTEGER PRIMARY KEY CHECK (id = 1),
  owner_id   TEXT NOT NULL,
  claimed_at TEXT NOT NULL
);

-- A device is keyed to the owner (authenticator user id). token_hash is the
-- SHA-256 of the hub-issued device token; the raw token is never stored.
CREATE TABLE IF NOT EXISTS devices (
  id           TEXT PRIMARY KEY,
  owner_id     TEXT NOT NULL,
  name         TEXT NOT NULL,
  kind         TEXT NOT NULL DEFAULT '',
  platform     TEXT NOT NULL DEFAULT '',
  capabilities TEXT NOT NULL DEFAULT '',  -- unit-separator (char 31) joined
  trust_state  TEXT NOT NULL,
  session_id   TEXT NOT NULL DEFAULT '',
  token_hash   TEXT NOT NULL DEFAULT '',
  last_seen_at TEXT NOT NULL,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_devices_owner_created
  ON devices(owner_id, created_at DESC);

-- token_hash is the lookup key for device-token trust enforcement (Phase 3).
CREATE INDEX IF NOT EXISTS idx_devices_token_hash
  ON devices(token_hash);

-- Pairing codes are short-TTL, single-use. code_hash is the SHA-256 of the raw
-- code; redeemed_at is empty until consumed. Expired/used codes are kept for a
-- short audit window and swept by the pairing purge (Phase 3 scheduler).
CREATE TABLE IF NOT EXISTS pairing_codes (
  code_hash    TEXT PRIMARY KEY,
  owner_id     TEXT NOT NULL,
  device_name  TEXT NOT NULL DEFAULT '',
  expires_at   TEXT NOT NULL,
  redeemed_at  TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pairing_codes_owner_expires
  ON pairing_codes(owner_id, expires_at DESC);
