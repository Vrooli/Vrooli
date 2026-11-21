CREATE TABLE IF NOT EXISTS relay_commands (
  command_id TEXT PRIMARY KEY,
  fingerprint TEXT NOT NULL,
  correlation_id TEXT NOT NULL,
  actor TEXT NOT NULL DEFAULT '',
  node_id TEXT NOT NULL,
  scenario TEXT NOT NULL,
  command TEXT NOT NULL,
  args_json TEXT NOT NULL,
  timeout_seconds INTEGER NOT NULL DEFAULT 0,
  max_response_bytes INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL,
  response_kind TEXT NOT NULL DEFAULT '',
  response_data BLOB,
  response_reason TEXT NOT NULL DEFAULT '',
  response_exit_code INTEGER NOT NULL DEFAULT 0,
  response_total_bytes INTEGER NOT NULL DEFAULT 0,
  route_name TEXT NOT NULL DEFAULT '',
  route_cost_units INTEGER NOT NULL DEFAULT 0,
  route_latency_ms INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_relay_commands_node ON relay_commands(node_id, updated_at);
