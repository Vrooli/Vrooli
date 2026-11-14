-- One row stores the operator policy and deterministic evaluator checkpoint.
-- The domain owns this table and this schema is applied through modules.AllSchemas.
CREATE TABLE IF NOT EXISTS lighting_policy_state (
  id TEXT PRIMARY KEY CHECK (id = 'default'),
  config_proto BLOB NOT NULL,
  evaluation_state TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
