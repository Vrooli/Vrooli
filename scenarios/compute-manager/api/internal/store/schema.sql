CREATE TABLE IF NOT EXISTS instance_intents (
  id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  requested_by TEXT NOT NULL DEFAULT '',
  provider TEXT NOT NULL,
  spec_json TEXT NOT NULL DEFAULT '{}',
  reservation_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'reserving',
  instance_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  resolved_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_instance_intents_state_created ON instance_intents(state, created_at);

CREATE TABLE IF NOT EXISTS instances (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  provider_instance_id TEXT NOT NULL,
  state TEXT NOT NULL,
  region TEXT NOT NULL DEFAULT '',
  size TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '',
  bridge_machine_id TEXT NOT NULL DEFAULT '',
  tenant TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '{}',
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  running_at TEXT NOT NULL DEFAULT '',
  destroyed_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_instances_state_expires ON instances(state, expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_provider_identity ON instances(provider, provider_instance_id);

CREATE TABLE IF NOT EXISTS provider_receipts (
  id TEXT PRIMARY KEY,
  intent_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  operation TEXT NOT NULL,
  provider_instance_id TEXT NOT NULL DEFAULT '',
  response_status TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reservations (
  id TEXT PRIMARY KEY,
  intent_id TEXT NOT NULL,
  instance_id TEXT NOT NULL DEFAULT '',
  supersedes TEXT NOT NULL DEFAULT '',
  meter_key TEXT NOT NULL,
  state TEXT NOT NULL,
  held_at TEXT NOT NULL,
  settled_at TEXT NOT NULL DEFAULT '',
  quantity INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_reservations_instance_state ON reservations(instance_id, state);

CREATE TABLE IF NOT EXISTS usage_records (
  id TEXT PRIMARY KEY,
  instance_id TEXT NOT NULL,
  tenant TEXT NOT NULL DEFAULT '',
  quantity INTEGER NOT NULL DEFAULT 0,
  started_at TEXT NOT NULL,
  ended_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_usage_records_instance_started ON usage_records(instance_id, started_at);

CREATE TABLE IF NOT EXISTS reconcile_findings (
  id TEXT PRIMARY KEY,
  observed_at TEXT NOT NULL,
  kind TEXT NOT NULL,
  provider TEXT NOT NULL,
  provider_instance_id TEXT NOT NULL DEFAULT '',
  instance_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  detail_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_reconcile_findings_status_observed ON reconcile_findings(status, observed_at);

CREATE TABLE IF NOT EXISTS bridge_key_cache (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  public_key TEXT NOT NULL,
  fingerprint TEXT NOT NULL DEFAULT '',
  key_type TEXT NOT NULL DEFAULT '',
  refreshed_at TEXT NOT NULL
);
