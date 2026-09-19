CREATE TABLE IF NOT EXISTS focus_sessions (
  id TEXT PRIMARY KEY,
  work_item_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  mode TEXT NOT NULL,
  state TEXT NOT NULL,
  started_at INTEGER NOT NULL,
  active_started_at INTEGER,
  ended_at INTEGER NOT NULL DEFAULT 0,
  active_seconds INTEGER NOT NULL DEFAULT 0,
  wall_seconds INTEGER NOT NULL DEFAULT 0,
  revision INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_focus_sessions_state ON focus_sessions(state);
CREATE UNIQUE INDEX IF NOT EXISTS idx_focus_one_current_session
  ON focus_sessions(state) WHERE state IN ('running', 'paused');

CREATE TABLE IF NOT EXISTS manual_actuals (
  id TEXT PRIMARY KEY,
  work_item_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  local_date TEXT NOT NULL,
  reported_minutes INTEGER NOT NULL,
  certainty TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  revision INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_manual_actuals_local_date ON manual_actuals(local_date);

CREATE TABLE IF NOT EXISTS actual_corrections (
  id TEXT PRIMARY KEY,
  actual_id TEXT NOT NULL,
  previous_minutes INTEGER NOT NULL,
  new_minutes INTEGER NOT NULL,
  previous_certainty TEXT NOT NULL,
  new_certainty TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);
