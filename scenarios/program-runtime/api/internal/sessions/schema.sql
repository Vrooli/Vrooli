CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_activity_at TEXT NOT NULL,
  sandbox_workspace TEXT NOT NULL DEFAULT '',
  memory_bytes INTEGER NOT NULL DEFAULT 0,
  reclaimed_reason TEXT NOT NULL DEFAULT '',
  inference_cost_micros INTEGER NOT NULL DEFAULT 0,
  inference_tokens INTEGER NOT NULL DEFAULT 0,
  delegation_cost_micros INTEGER NOT NULL DEFAULT 0,
  inference_ceiling_micros INTEGER NOT NULL DEFAULT 0,
  delegation_ceiling_micros INTEGER NOT NULL DEFAULT 0,
  delegation_spend_measured INTEGER NOT NULL DEFAULT 0,
  delegation_spend_note TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity_at);

CREATE TABLE IF NOT EXISTS session_grants (
  session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  grant_name TEXT NOT NULL,
  PRIMARY KEY (session_id, grant_name)
);

CREATE INDEX IF NOT EXISTS idx_session_grants_name ON session_grants(grant_name);

CREATE TABLE IF NOT EXISTS reclamation_reasons (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  reason TEXT NOT NULL,
  reclaimed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_reclamation_reasons_session ON reclamation_reasons(session_id);
CREATE INDEX IF NOT EXISTS idx_reclamation_reasons_time ON reclamation_reasons(reclaimed_at);
