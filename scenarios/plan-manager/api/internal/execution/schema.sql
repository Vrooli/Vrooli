-- executions / handoffs / findings / velocity_points — owned by internal/execution/.
-- The guided-runner store (docs/concepts/DATA.md): run↔plan linkage, captured
-- decisions/findings, candidate findings, canonical handoff records, and the
-- per-plan velocity series.
--
-- Unlike the plans store (a single plan document), these rows ARE queried across
-- one another — findings by execution and triage state, velocity by plan, the
-- handoff by execution — so each is a first-class table with queryable columns
-- plus a `document` JSON column for the structured payload that round-trips with
-- the row (e.g. the handoff's assembled decisions/findings snapshot). Phases and
-- references stay owned by the plans domain and are never copied here.
--
-- Embedded by schema.go and applied idempotently via database.EnsureSchemas at
-- boot through the modules.AllSchemas registry. Timestamps are RFC3339Nano. Use
-- CREATE ... IF NOT EXISTS so re-runs are no-ops. Pool=1 SQLite: callers collect
-- rows into slices before any follow-up query (no nested query inside an open
-- rows loop).

-- executions — run↔plan linkage and the runner's current-phase pointer.
CREATE TABLE IF NOT EXISTS executions (
  id               TEXT PRIMARY KEY,
  plan_id          TEXT NOT NULL,
  run_id           TEXT NOT NULL DEFAULT '',
  current_phase_id TEXT NOT NULL DEFAULT '',
  complete         INTEGER NOT NULL DEFAULT 0,
  started_at       TEXT NOT NULL,
  updated_at       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_executions_plan ON executions(plan_id);
CREATE INDEX IF NOT EXISTS idx_executions_run ON executions(run_id);

-- decisions — in-flow design decisions captured during a run, FK→execution.
CREATE TABLE IF NOT EXISTS decisions (
  id           TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL,
  phase_id     TEXT NOT NULL DEFAULT '',
  summary      TEXT NOT NULL DEFAULT '',
  detail       TEXT NOT NULL DEFAULT '',
  recorded_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_decisions_execution ON decisions(execution_id);

-- findings — candidate (unvalidated) findings, FK→execution, with triage state.
-- attribution_run_id powers attribution-keyed dedup; the unique index makes the
-- (execution_id, attribution_run_id, title) dedup key authoritative at the store
-- when an attribution run id is present (best-effort dedup-by-title otherwise).
CREATE TABLE IF NOT EXISTS findings (
  id                 TEXT PRIMARY KEY,
  execution_id       TEXT NOT NULL,
  phase_id           TEXT NOT NULL DEFAULT '',
  title              TEXT NOT NULL DEFAULT '',
  detail             TEXT NOT NULL DEFAULT '',
  triage             TEXT NOT NULL DEFAULT 'candidate',
  attribution_run_id TEXT NOT NULL DEFAULT '',
  recorded_at        TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_findings_execution ON findings(execution_id);
CREATE INDEX IF NOT EXISTS idx_findings_triage ON findings(triage);

-- The (execution_id, attribution_run_id, title) dedup key, now authoritative at
-- the store. PARTIAL (only rows that carry an attribution run id) so two
-- concurrent CLI processes recording the same finding cannot both insert, while
-- attribution-less findings still dedup best-effort by title in the service
-- without a hard constraint. This is the index the comment above promised.
CREATE UNIQUE INDEX IF NOT EXISTS idx_findings_dedup
  ON findings(execution_id, attribution_run_id, title)
  WHERE attribution_run_id <> '';

-- handoffs — canonical structured handoff records, FK→execution. The assembled
-- decisions/candidate-findings snapshot and the last-validation result live in
-- the `document` JSON column; the queryable columns power lookup by execution.
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
