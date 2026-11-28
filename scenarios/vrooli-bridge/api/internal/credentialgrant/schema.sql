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
  revision INTEGER NOT NULL DEFAULT 1,
  granted_at TEXT NOT NULL,
  revoked_at TEXT NOT NULL DEFAULT '',
  acked_generation INTEGER NOT NULL DEFAULT 0,
  receipt_at TEXT NOT NULL DEFAULT '',
  receipt_accepted INTEGER NOT NULL DEFAULT 0,
  receipt_reason TEXT NOT NULL DEFAULT '',
  purge_state TEXT NOT NULL DEFAULT '',
  purge_receipt_at TEXT NOT NULL DEFAULT '',
  purge_accepted INTEGER NOT NULL DEFAULT 0,
  purge_reason TEXT NOT NULL DEFAULT '',
  UNIQUE (node_id, logical_id, field)
);

CREATE INDEX IF NOT EXISTS idx_credential_grants_node ON credential_grants(node_id, revoked_at);
CREATE INDEX IF NOT EXISTS idx_credential_grants_address ON credential_grants(logical_id, field, revoked_at);

CREATE TABLE IF NOT EXISTS credential_grant_mutation_receipts (
  actor_id TEXT NOT NULL,
  request_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  node_id TEXT NOT NULL,
  enrollment_generation INTEGER NOT NULL,
  subject TEXT NOT NULL,
  expected_revision INTEGER NOT NULL,
  result_revision INTEGER NOT NULL,
  payload_digest TEXT NOT NULL,
  grant_id TEXT NOT NULL,
  outcome TEXT NOT NULL,
  delivery_state TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL,
  PRIMARY KEY (request_id)
);
