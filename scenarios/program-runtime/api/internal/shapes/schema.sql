CREATE TABLE IF NOT EXISTS program_shapes (
  shape_key TEXT PRIMARY KEY,
  binding_ids TEXT NOT NULL,
  binding_count INTEGER NOT NULL,
  occurrences INTEGER NOT NULL DEFAULT 0,
  agent_runs INTEGER NOT NULL DEFAULT 0,
  operator_runs INTEGER NOT NULL DEFAULT 0,
  test_runs INTEGER NOT NULL DEFAULT 0,
  replay_runs INTEGER NOT NULL DEFAULT 0,
  sessions INTEGER NOT NULL DEFAULT 0,
  first_seen TEXT NOT NULL,
  last_seen TEXT NOT NULL,
  exemplar_program_id TEXT NOT NULL,
  exemplar_bytes INTEGER NOT NULL DEFAULT 0,
  covered_by TEXT NOT NULL DEFAULT '',
  covered_reason TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'observed' CHECK (state IN ('observed','nominated','covered')),
  CHECK (binding_count >= 2)
);
CREATE INDEX IF NOT EXISTS idx_program_shapes_state ON program_shapes(state, occurrences DESC);
CREATE INDEX IF NOT EXISTS idx_program_shapes_last_seen ON program_shapes(last_seen);

CREATE TABLE IF NOT EXISTS program_shape_observations (
  program_id TEXT PRIMARY KEY,
  shape_key TEXT NOT NULL,
  observed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS program_shape_sessions (
  shape_key TEXT NOT NULL,
  session_id TEXT NOT NULL,
  PRIMARY KEY (shape_key, session_id)
);
