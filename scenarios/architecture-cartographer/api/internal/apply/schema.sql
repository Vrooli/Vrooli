-- Apply tables — owned by internal/apply/. v0.1 ships the table shape
-- so v0.2's apply execution can persist plans + runs without a
-- migration. The plan operations are stored as a JSON payload (same
-- pattern as conflicts) so v0.2 can extend OperationKind without
-- changing the column set.

CREATE TABLE IF NOT EXISTS apply_plans (
  id         TEXT PRIMARY KEY,
  scenario   TEXT NOT NULL,
  domain     TEXT NOT NULL,
  payload    BLOB,
  planned_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_apply_plans_scenario_domain
  ON apply_plans(scenario, domain, planned_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS apply_runs (
  id          TEXT PRIMARY KEY,
  plan_id     TEXT NOT NULL,
  scenario    TEXT NOT NULL,
  domain      TEXT NOT NULL,
  status      TEXT NOT NULL,
  build_log   TEXT NOT NULL DEFAULT '',
  started_at  TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_apply_runs_scenario_time
  ON apply_runs(scenario, started_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS apply_baselines (
  scenario     TEXT PRIMARY KEY,
  green        INTEGER NOT NULL DEFAULT 0,
  toolchain    TEXT NOT NULL DEFAULT '',
  log          TEXT NOT NULL DEFAULT '',
  captured_at  TEXT NOT NULL
);
