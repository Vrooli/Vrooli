package credentialgrant

const schemaSQL = `
CREATE TABLE IF NOT EXISTS credential_generations (
  logical_id TEXT NOT NULL,
  field TEXT NOT NULL,
  generation INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (logical_id, field)
);
CREATE TABLE IF NOT EXISTS credential_grants (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  logical_id TEXT NOT NULL,
  field TEXT NOT NULL,
  class TEXT NOT NULL,
  retention TEXT NOT NULL,
  generation INTEGER NOT NULL DEFAULT 1,
  granted_at TEXT NOT NULL,
  revoked_at TEXT NOT NULL DEFAULT '',
  acked_generation INTEGER NOT NULL DEFAULT 0,
  receipt_at TEXT NOT NULL DEFAULT '',
  receipt_accepted INTEGER NOT NULL DEFAULT 0,
  receipt_reason TEXT NOT NULL DEFAULT '',
  UNIQUE (node_id, logical_id, field)
);
CREATE INDEX IF NOT EXISTS idx_credential_grants_node ON credential_grants(node_id, revoked_at);
CREATE INDEX IF NOT EXISTS idx_credential_grants_address ON credential_grants(logical_id, field, revoked_at);
`

func Schema() string { return schemaSQL }
