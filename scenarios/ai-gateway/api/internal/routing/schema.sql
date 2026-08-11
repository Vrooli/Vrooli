CREATE TABLE IF NOT EXISTS route_events (
  event_id TEXT PRIMARY KEY,
  request_id TEXT NOT NULL,
  scenario TEXT NOT NULL,
  operation TEXT NOT NULL,
  role TEXT NOT NULL,
  profile INTEGER NOT NULL,
  privacy_class INTEGER NOT NULL,
  selected_provider TEXT NOT NULL,
  selected_locality TEXT NOT NULL,
  status TEXT NOT NULL,
  policy_reasons_json TEXT NOT NULL,
  failure_reasons_json TEXT NOT NULL,
  fallback_used INTEGER NOT NULL DEFAULT 0,
  prompt_redacted INTEGER NOT NULL DEFAULT 1,
  response_redacted INTEGER NOT NULL DEFAULT 1,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  breaker_state TEXT NOT NULL DEFAULT '',
  failure_class TEXT NOT NULL DEFAULT '',
  rejection_reason TEXT NOT NULL DEFAULT '',
  capacity_verdict TEXT NOT NULL DEFAULT '',
  capacity_claim_id TEXT NOT NULL DEFAULT '',
  capacity_required_bytes INTEGER NOT NULL DEFAULT 0,
  capacity_granted_bytes INTEGER NOT NULL DEFAULT 0,
  capacity_reclaim_required INTEGER NOT NULL DEFAULT 0,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cost_estimate REAL NOT NULL DEFAULT 0,
  selected_model TEXT NOT NULL DEFAULT '',
  image_count INTEGER NOT NULL DEFAULT 0,
  attachment_bytes INTEGER NOT NULL DEFAULT 0,
  attachment_sha256 TEXT NOT NULL DEFAULT '',
  attachments_redacted INTEGER NOT NULL DEFAULT 1,
  attachment_dimensions_json TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_route_events_created_at ON route_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_route_events_scenario ON route_events(scenario, created_at DESC);

CREATE TABLE IF NOT EXISTS provider_health (
  provider TEXT NOT NULL,
  role TEXT NOT NULL,
  kind INTEGER NOT NULL,
  state TEXT NOT NULL DEFAULT 'closed',
  consecutive_failures INTEGER NOT NULL DEFAULT 0,
  last_failure_class TEXT NOT NULL DEFAULT '',
  last_success_at TEXT NOT NULL DEFAULT '',
  last_failure_at TEXT NOT NULL DEFAULT '',
  cooldown_until TEXT NOT NULL DEFAULT '',
  opened_at TEXT NOT NULL DEFAULT '',
  generation INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (provider, role, kind)
);

CREATE TABLE IF NOT EXISTS media_executions (
  execution_id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  status INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  started_at TEXT NOT NULL DEFAULT '',
  completed_at TEXT NOT NULL DEFAULT '',
  request_json TEXT NOT NULL,
  route_evidence_json TEXT NOT NULL DEFAULT '',
  outputs_json TEXT NOT NULL DEFAULT '',
  actual_cost_usd REAL NOT NULL DEFAULT 0,
  resolved_model TEXT NOT NULL DEFAULT '',
  seed TEXT NOT NULL DEFAULT '',
  warnings_json TEXT NOT NULL DEFAULT '',
  error_code TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_media_executions_created_at ON media_executions(created_at DESC);
