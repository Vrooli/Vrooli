-- Snapshot domain — owned by internal/snapshot/. Embedded by schema.go and
-- applied through modules.AllSchemas at boot.

CREATE TABLE IF NOT EXISTS network_snapshots (
  id            TEXT PRIMARY KEY,
  status        TEXT NOT NULL,
  profile       TEXT NOT NULL,
  summary       TEXT NOT NULL,
  findings_json TEXT NOT NULL DEFAULT '[]',
  created_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_network_snapshots_created
  ON network_snapshots(created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS snapshot_probe_results (
  id          TEXT PRIMARY KEY,
  snapshot_id TEXT NOT NULL REFERENCES network_snapshots(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  value       TEXT NOT NULL,
  unit        TEXT NOT NULL,
  status      TEXT NOT NULL,
  sort_order  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_snapshot_probe_results_snapshot_order
  ON snapshot_probe_results(snapshot_id, sort_order ASC);
