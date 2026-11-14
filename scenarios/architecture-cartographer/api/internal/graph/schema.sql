-- Graph snapshot cache — owned by internal/graph/. The snapshot itself
-- is stored as canonical-form JSON so the proto remains the
-- source-of-truth for the wire/storage shape (rather than reifying every
-- node/edge into per-row tables). Lookups are by id,
-- (scenario, content_hash), or the latest non-degraded
-- (scenario, source_fingerprint).

CREATE TABLE IF NOT EXISTS graph_snapshots (
  id            TEXT PRIMARY KEY,
  scenario      TEXT NOT NULL,
  content_hash  TEXT NOT NULL,
  source_fingerprint TEXT NOT NULL DEFAULT '',
  payload       BLOB NOT NULL,
  -- Encoding of payload: empty for legacy raw JSON, 'gzip' for compressed.
  payload_codec TEXT NOT NULL DEFAULT '',
  extracted_at  TEXT NOT NULL,
  extraction_ms INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_graph_snapshots_scenario_time
  ON graph_snapshots(scenario, extracted_at DESC, id DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_graph_snapshots_scenario_hash
  ON graph_snapshots(scenario, content_hash);

-- The source-fingerprint index is installed by the graph repository's lazy
-- migration. Keeping it out of startup schema lets existing databases that
-- predate the column boot; the repository adds the column before creating the
-- index on first graph access.
