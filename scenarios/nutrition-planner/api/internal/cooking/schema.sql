CREATE TABLE IF NOT EXISTS cooking_sessions (
  workspace_id TEXT NOT NULL,
  session_id TEXT NOT NULL,
  recipe_id TEXT NOT NULL,
  recipe_revision INTEGER NOT NULL,
  method_id TEXT NOT NULL,
  scale TEXT NOT NULL,
  current_step_index INTEGER NOT NULL DEFAULT 0,
  completed_steps_json TEXT NOT NULL DEFAULT '[]',
  timers_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL DEFAULT 'active',
  actual_yield TEXT NOT NULL DEFAULT '',
  yield_unit TEXT NOT NULL DEFAULT '',
  version INTEGER NOT NULL DEFAULT 1,
  started_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(workspace_id, session_id)
);
CREATE INDEX IF NOT EXISTS idx_cooking_sessions_workspace_status ON cooking_sessions(workspace_id,status,updated_at DESC);
CREATE TABLE IF NOT EXISTS cooking_session_events (
  workspace_id TEXT NOT NULL,
  session_id TEXT NOT NULL,
  event_id TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, session_id, event_id),
  FOREIGN KEY(workspace_id,session_id) REFERENCES cooking_sessions(workspace_id,session_id)
);
