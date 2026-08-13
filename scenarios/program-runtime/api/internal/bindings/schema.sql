CREATE TABLE IF NOT EXISTS refusals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  binding_id TEXT NOT NULL,
  reason TEXT NOT NULL,
  occurred_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_refusals_session ON refusals(session_id);
CREATE INDEX IF NOT EXISTS idx_refusals_binding ON refusals(binding_id);
CREATE INDEX IF NOT EXISTS idx_refusals_occurred ON refusals(occurred_at);

CREATE TABLE IF NOT EXISTS unresolved_binding_attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  attempted_name TEXT NOT NULL,
  occurred_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_unresolved_binding_attempts_name ON unresolved_binding_attempts(attempted_name);
CREATE INDEX IF NOT EXISTS idx_unresolved_binding_attempts_occurred ON unresolved_binding_attempts(occurred_at);

CREATE TABLE IF NOT EXISTS binding_invocations (
  invocation_id TEXT PRIMARY KEY,
  binding_id TEXT NOT NULL,
  target_scenario TEXT NOT NULL,
  session_id TEXT NOT NULL,
  program_id TEXT NOT NULL,
  provenance TEXT NOT NULL,
  outcome TEXT NOT NULL CHECK (outcome IN ('success', 'refused', 'failed')),
  reason TEXT NOT NULL DEFAULT '',
  latency_ms INTEGER NOT NULL,
  usage_input_tokens INTEGER NOT NULL DEFAULT 0,
  usage_output_tokens INTEGER NOT NULL DEFAULT 0,
  usage_cost_micros INTEGER NOT NULL DEFAULT 0,
  origin TEXT NOT NULL DEFAULT 'organic',
  invocation_class TEXT NOT NULL DEFAULT 'success',
  occurred_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_binding_invocations_binding ON binding_invocations(binding_id);
CREATE INDEX IF NOT EXISTS idx_binding_invocations_occurred ON binding_invocations(occurred_at);
CREATE INDEX IF NOT EXISTS idx_binding_invocations_outcome ON binding_invocations(outcome);
