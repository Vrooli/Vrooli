-- Artifact distributions — owned by internal/artifacts/. Embedded by schema.go
-- and applied via database.EnsureSchemas at boot (api/main.go through
-- modules.AllSchemas). Non-git artifact distribution (OT-P1-003): one durable
-- Distribution record per delivery, holding the device-sync-hub reference +
-- metadata only — never the bytes (DATA.md). Times are RFC3339Nano strings
-- matching the wire format and the time.Time round-trip in sqlite.go. Use CREATE
-- TABLE IF NOT EXISTS so re-runs are no-ops (migrate, never recreate). Postgres-
-- compatible column types for forward scale.
CREATE TABLE IF NOT EXISTS distributions (
  id               TEXT PRIMARY KEY,
  node_id          TEXT NOT NULL,
  name             TEXT NOT NULL DEFAULT '',
  source_ref       TEXT NOT NULL DEFAULT '',
  destination_path TEXT NOT NULL DEFAULT '',
  status           INTEGER NOT NULL DEFAULT 0,
  delivery_ref     TEXT NOT NULL DEFAULT '',
  detail           TEXT NOT NULL DEFAULT '',
  created_at       TEXT NOT NULL,
  updated_at       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_distributions_created_at ON distributions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_distributions_node_id ON distributions(node_id);

-- Produced run artifacts are bounded evidence bytes (for example a screenshot),
-- distinct from large artifact distribution. They are owned by the run and may
-- only be written by that run's authenticated node.
CREATE TABLE IF NOT EXISTS produced_artifacts (
  run_id       TEXT NOT NULL,
  name         TEXT NOT NULL,
  media_type   TEXT NOT NULL DEFAULT 'application/octet-stream',
  data         BLOB NOT NULL,
  size_bytes   INTEGER NOT NULL,
  artifact_ref TEXT NOT NULL,
  created_at   TEXT NOT NULL,
  PRIMARY KEY (run_id, name)
);

CREATE INDEX IF NOT EXISTS idx_produced_artifacts_created_at ON produced_artifacts(created_at DESC);
