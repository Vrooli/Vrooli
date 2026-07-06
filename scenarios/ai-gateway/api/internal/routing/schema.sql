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
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_route_events_created_at ON route_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_route_events_scenario ON route_events(scenario, created_at DESC);
