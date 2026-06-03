-- Runs + per-target outcomes — owned by internal/runs/. Run history is bounded
-- catalog data (a cache + the last-success/last-verified anchor), not the
-- source of truth. Times are RFC3339Nano strings. CREATE ... IF NOT EXISTS so
-- re-runs are no-ops.
CREATE TABLE IF NOT EXISTS runs (
  id          TEXT PRIMARY KEY,
  plan_id     TEXT NOT NULL,
  trigger     TEXT NOT NULL,
  status      TEXT NOT NULL,
  started_at  TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT '',
  -- run-level failure reason (plan-resolution failure, startup reconciliation);
  -- per-target errors live on run_outcomes.error
  error       TEXT NOT NULL DEFAULT '',
  -- heartbeat: last time the run's status/outcomes were persisted, so an
  -- in-flight or wedged run is observable and startup reconciliation can age it
  updated_at  TEXT NOT NULL DEFAULT '',
  -- physical (deduped+compressed) repo growth attributable to this run, summed
  -- across the destinations it wrote — the on-disk cost vs the logical bytes on
  -- run_outcomes. A repo-size delta measured around the run; approximate when
  -- runs to the same repo overlap (see internal/runs/stats.go).
  physical_bytes INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_runs_plan ON runs(plan_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_started ON runs(started_at DESC);

CREATE TABLE IF NOT EXISTS run_outcomes (
  run_id         TEXT NOT NULL,
  target_id      TEXT NOT NULL,
  destination_id TEXT NOT NULL,
  status         TEXT NOT NULL,
  snapshot_id    TEXT NOT NULL DEFAULT '',
  bytes          INTEGER NOT NULL DEFAULT 0,
  error          TEXT NOT NULL DEFAULT '',
  started_at     TEXT NOT NULL,
  finished_at    TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (run_id, target_id, destination_id),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_outcomes_target ON run_outcomes(target_id, started_at DESC);
