-- Jobs table — owned by internal/jobs/. Embedded by schema.go and applied via
-- database.EnsureSchemas at boot through the modules.AllSchemas registry. Times
-- are RFC3339Nano strings (matching the time.Time round-trip in store.go).
-- Server-owned durable jobs: rows persist across client disconnects and survive
-- restarts (recovery marks orphaned non-terminal jobs as failed/interrupted).
CREATE TABLE IF NOT EXISTS jobs (
  id                TEXT PRIMARY KEY,
  operation         TEXT NOT NULL,
  lane              TEXT NOT NULL,
  state             TEXT NOT NULL,
  progress          INTEGER NOT NULL DEFAULT 0,
  message           TEXT NOT NULL DEFAULT '',
  error             TEXT NOT NULL DEFAULT '',
  result_ref        TEXT NOT NULL DEFAULT '',
  payload           BLOB,
  estimated_seconds INTEGER NOT NULL DEFAULT 0,
  created_at        TEXT NOT NULL,
  started_at        TEXT NOT NULL DEFAULT '',
  finished_at       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at DESC);
