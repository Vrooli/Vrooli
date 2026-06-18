-- Runs + run_events tables — owned by internal/runs/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot (api/main.go through
-- modules.AllSchemas). Durable, server-owned remote runs (OT-P0-005) and their
-- append-only event history. Times are RFC3339Nano strings matching the wire
-- format and the time.Time round-trip in sqlite.go. Args/artifact_refs are
-- JSON-encoded string arrays. Use CREATE TABLE IF NOT EXISTS so re-runs are
-- no-ops (migrate, never recreate). Postgres-compatible column types for
-- forward scale.
CREATE TABLE IF NOT EXISTS runs (
  id              TEXT PRIMARY KEY,
  node_id         TEXT NOT NULL,
  scenario        TEXT NOT NULL DEFAULT '',
  verb            TEXT NOT NULL DEFAULT '',
  args            TEXT NOT NULL DEFAULT '[]',
  status          INTEGER NOT NULL DEFAULT 0,
  exit_code       INTEGER NOT NULL DEFAULT 0,
  timeout_seconds INTEGER NOT NULL DEFAULT 0,
  created_at      TEXT NOT NULL,
  started_at      TEXT NOT NULL DEFAULT '',
  finished_at     TEXT NOT NULL DEFAULT '',
  artifact_refs   TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_node_id ON runs(node_id);

-- run_events is append-only: rows are only ever INSERTed and SELECTed (there is
-- no UPDATE/DELETE in sqlite.go). The (run_id, sequence) pair is unique so a
-- node re-sending an event (at-least-once delivery) is de-duplicated rather than
-- double-stored.
CREATE TABLE IF NOT EXISTS run_events (
  run_id       TEXT NOT NULL,
  sequence     INTEGER NOT NULL,
  kind         INTEGER NOT NULL DEFAULT 0,
  log_chunk    TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT '',
  exit_code    INTEGER NOT NULL DEFAULT 0,
  artifact_ref TEXT NOT NULL DEFAULT '',
  emitted_at   TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (run_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_run_events_run ON run_events(run_id, sequence);
