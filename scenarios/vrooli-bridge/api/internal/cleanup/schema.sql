CREATE TABLE IF NOT EXISTS cleanup_operations (
  id TEXT PRIMARY KEY,
  machine_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  target TEXT NOT NULL,
  scope TEXT NOT NULL,
  status INTEGER NOT NULL,
  transport TEXT NOT NULL DEFAULT '',
  transport_reason TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  plan_hash TEXT NOT NULL DEFAULT '',
  plan_json BLOB,
  receipt_json BLOB,
  operator_id TEXT NOT NULL DEFAULT '',
  sealed_passphrase BLOB,
  capability BLOB,
  sealing_public_key BLOB,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS cleanup_machine_active_idx
  ON cleanup_operations(machine_id)
  WHERE status IN (1, 2, 3, 4, 5);

CREATE TABLE IF NOT EXISTS cleanup_events (
  operation_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  kind INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT '',
  log_chunk TEXT NOT NULL DEFAULT '',
  plan_json BLOB,
  receipt_json BLOB,
  reason TEXT NOT NULL DEFAULT '',
  exit_code INTEGER NOT NULL DEFAULT 0,
  emitted_at TEXT NOT NULL,
  PRIMARY KEY(operation_id, sequence),
  FOREIGN KEY(operation_id) REFERENCES cleanup_operations(id) ON DELETE CASCADE
);
