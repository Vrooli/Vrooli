-- verification_runs — owned by internal/runs/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot (api/main.go through the
-- modules.AllSchemas registry). Times are stored as RFC3339Nano strings
-- matching the wire format and the time.Time round-trip in
-- sqlite.go::scanRun. Use CREATE TABLE IF NOT EXISTS so re-runs are
-- no-ops.
CREATE TABLE IF NOT EXISTS verification_runs (
  id             TEXT PRIMARY KEY,
  flow_id        TEXT NOT NULL,
  flow_path      TEXT NOT NULL,
  root           TEXT NOT NULL,
  source_sha256  TEXT NOT NULL DEFAULT '',
  model_sha256   TEXT NOT NULL DEFAULT '',
  gen_sha256     TEXT NOT NULL DEFAULT '',
  mode           TEXT NOT NULL,
  status         TEXT NOT NULL CHECK (status IN ('passed','failed','error')),
  counterexample TEXT,
  error_message  TEXT NOT NULL DEFAULT '',
  output         TEXT NOT NULL DEFAULT '',
  started_at     TEXT NOT NULL,
  finished_at    TEXT NOT NULL,
  duration_ms    INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runs_flow_id
  ON verification_runs (flow_id, finished_at DESC);

CREATE INDEX IF NOT EXISTS idx_runs_finished_at
  ON verification_runs (finished_at DESC);
