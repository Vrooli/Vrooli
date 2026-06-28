-- executions / handoffs / velocity_points — owned by internal/execution/.
-- The guided-runner store (docs/concepts/DATA.md): run↔plan linkage, canonical
-- handoff records, and the per-plan velocity series.
--
-- Unlike the plans store (a single plan document), these rows ARE queried across
-- one another — velocity by plan, the handoff by execution — so each is a
-- first-class table with queryable columns plus a `document` JSON column for the
-- structured payload that round-trips with the row (the handoff's assembled log
-- summary snapshot). Decisions/findings/bugs/records live in the log domain;
-- phases and references stay owned by the plans domain; neither is copied here.
--
-- Embedded by schema.go and applied idempotently via database.EnsureSchemas at
-- boot through the modules.AllSchemas registry. Timestamps are RFC3339Nano. Use
-- CREATE ... IF NOT EXISTS so re-runs are no-ops. Pool=1 SQLite: callers collect
-- rows into slices before any follow-up query (no nested query inside an open
-- rows loop).

-- executions — run↔plan linkage and the runner's current-phase pointer.
-- inputs_freshened_at/freshen_status/freshen_detail record the one-time
-- execution-start "freshen inputs" step (baseline snapshot capture + reference
-- staleness recompute, delegated to the validation domain). They stay empty until
-- the first start/resume freshens, and are re-attempted while status != 'captured'.
CREATE TABLE IF NOT EXISTS executions (
  id                  TEXT PRIMARY KEY,
  plan_id             TEXT NOT NULL,
  run_id              TEXT NOT NULL DEFAULT '',
  current_phase_id    TEXT NOT NULL DEFAULT '',
  complete            INTEGER NOT NULL DEFAULT 0,
  started_at          TEXT NOT NULL,
  updated_at          TEXT NOT NULL,
  inputs_freshened_at TEXT NOT NULL DEFAULT '',
  freshen_status      TEXT NOT NULL DEFAULT '',
  freshen_detail      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_executions_plan ON executions(plan_id);
CREATE INDEX IF NOT EXISTS idx_executions_run ON executions(run_id);

-- Decisions and candidate findings are NOT stored here — they are typed entries
-- in the log domain's log_entries table (internal/planlog). The handoff snapshots
-- a compact log summary read from that domain.

-- handoffs — canonical structured handoff records, FK→execution. The assembled
-- log-ledger snapshot and the last-validation result live in the `document` JSON
-- column; the queryable columns power lookup by execution.
CREATE TABLE IF NOT EXISTS handoffs (
  id             TEXT PRIMARY KEY,
  execution_id   TEXT NOT NULL,
  plan_id        TEXT NOT NULL,
  completeness   TEXT NOT NULL DEFAULT '',
  resume_phase_id TEXT NOT NULL DEFAULT '',
  document       TEXT NOT NULL DEFAULT '{}',
  assembled_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_handoffs_execution ON handoffs(execution_id);

-- velocity_points — per-plan/run time/tokens/iterations series. Queried by plan.
CREATE TABLE IF NOT EXISTS velocity_points (
  id                TEXT PRIMARY KEY,
  plan_id           TEXT NOT NULL,
  run_id            TEXT NOT NULL DEFAULT '',
  wall_time_seconds INTEGER NOT NULL DEFAULT 0,
  tokens            INTEGER NOT NULL DEFAULT 0,
  iterations        INTEGER NOT NULL DEFAULT 0,
  completeness      TEXT NOT NULL DEFAULT '',
  recorded_at       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_velocity_plan ON velocity_points(plan_id);
