-- trials_runs + trial_gates — owned by internal/trials/. The empirical-gate
-- history: one row per dispatched trial (verdict + tokens + wall-time +
-- sandbox-diff attribution) retained indefinitely (the efficiency trend over
-- time IS the value), and the per-Guide-task gate registry (how many runs back
-- each Guide task, for gate-coverage). Embedded by schema.go and applied via
-- database.EnsureSchemas at boot through the modules.AllSchemas registry.
-- created_at/updated_at are RFC3339Nano (see sqlite.go). Use CREATE TABLE IF NOT
-- EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS trials_runs (
  id               TEXT PRIMARY KEY,
  task_id          TEXT NOT NULL,
  suite            TEXT NOT NULL,
  model            TEXT NOT NULL DEFAULT '',
  guide_task_id    TEXT NOT NULL DEFAULT '',
  verdict          INTEGER NOT NULL DEFAULT 0,
  tokens           INTEGER NOT NULL DEFAULT 0,
  duration_ms      INTEGER NOT NULL DEFAULT 0,
  sandbox_diff_ref TEXT NOT NULL DEFAULT '',
  created_at       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trials_runs_created ON trials_runs(created_at);
CREATE INDEX IF NOT EXISTS idx_trials_runs_suite ON trials_runs(suite);
CREATE INDEX IF NOT EXISTS idx_trials_runs_task ON trials_runs(task_id);

CREATE TABLE IF NOT EXISTS trial_gates (
  task_key   TEXT PRIMARY KEY,
  gate_count INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
