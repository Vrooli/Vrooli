CREATE TABLE IF NOT EXISTS persona_channels (
  id TEXT PRIMARY KEY NOT NULL,
  persona_id TEXT NOT NULL REFERENCES personas(id),
  kind TEXT NOT NULL CHECK (kind IN ('email', 'sms', 'device')),
  address TEXT NOT NULL,
  credential_ref TEXT NOT NULL DEFAULT '',
  adapter TEXT NOT NULL,
  enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
  created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS persona_channels_one_enabled_idx ON persona_channels(persona_id) WHERE enabled = 1;
