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
  failure_code TEXT NOT NULL DEFAULT '',
  failure_category TEXT NOT NULL DEFAULT '',
  next_action TEXT NOT NULL DEFAULT '',
  -- heartbeat: last time the run's status/outcomes were persisted, so an
  -- in-flight or wedged run is observable and startup reconciliation can age it
  updated_at  TEXT NOT NULL DEFAULT '',
  -- physical (deduped+compressed) repo growth attributable to this run, summed
  -- across the destinations it wrote — the on-disk cost vs the logical bytes on
  -- run_outcomes. A repo-size delta measured around the run; approximate when
  -- runs to the same repo overlap (see internal/runs/stats.go).
  physical_bytes INTEGER NOT NULL DEFAULT 0
);

-- Close the check-then-insert race in TriggerRun. The service-level check
-- provides a friendly conflict for normal calls; this index is authoritative
-- when two callers arrive concurrently.
CREATE UNIQUE INDEX IF NOT EXISTS runs_one_active_plan
  ON runs(plan_id)
  WHERE status IN ('pending', 'capturing', 'snapshotting');

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
  failure_code   TEXT NOT NULL DEFAULT '',
  failure_category TEXT NOT NULL DEFAULT '',
  warning        TEXT NOT NULL DEFAULT '',
  started_at     TEXT NOT NULL,
  finished_at    TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (run_id, target_id, destination_id),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_outcomes_target ON run_outcomes(target_id, started_at DESC);

CREATE TABLE IF NOT EXISTS run_incidents (
  run_id TEXT NOT NULL,
  code TEXT NOT NULL,
  category TEXT NOT NULL,
  scope TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  next_action TEXT NOT NULL DEFAULT '',
  destination_id TEXT NOT NULL DEFAULT '',
  target_ids TEXT NOT NULL DEFAULT '[]',
  first_observed TEXT NOT NULL DEFAULT '',
  last_observed TEXT NOT NULL DEFAULT '',
  last_known_good TEXT NOT NULL DEFAULT '',
  retryable INTEGER NOT NULL DEFAULT 0,
  retry_after_seconds INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (run_id, code, scope, destination_id),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);
