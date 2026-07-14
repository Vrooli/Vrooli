-- Adoptions domain schema — owned by internal/adoptions/.
-- Applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry. Idempotent: CREATE TABLE/INDEX IF NOT
-- EXISTS so re-runs are no-ops.
--
-- `adoption_records` is the soft link from a library component to a
-- concrete copy of its source under a target scenario's tree.
-- component_id is a soft FK to components.id — no REFERENCES
-- constraint, per storage-steer per-domain rule. library_id is echoed
-- at create time so rows survive component_id reassignment / removal.
CREATE TABLE IF NOT EXISTS adoption_records (
  id              TEXT PRIMARY KEY,
  component_id    TEXT NOT NULL,
  library_id      TEXT NOT NULL DEFAULT '',
  scenario        TEXT NOT NULL,
  adopted_path    TEXT NOT NULL,
  adopted_version TEXT NOT NULL DEFAULT '',
  source_sha256   TEXT NOT NULL DEFAULT '',
  adopted_snapshot_sha256 TEXT NOT NULL DEFAULT '',
  library_version_status TEXT NOT NULL DEFAULT '',
  local_status    TEXT NOT NULL DEFAULT '',
  status_detail   TEXT NOT NULL DEFAULT '',
  created_at      TEXT NOT NULL,
  refreshed_at    TEXT NOT NULL DEFAULT '',
  applied_at      TEXT NOT NULL DEFAULT '',
  -- drift_backlog_ref records the swarm-manager backlog item
  -- (`<kind>/<name>`) filed when this adoption first transitioned to
  -- behind/modified. Cleared back to '' when status returns to
  -- current, so a subsequent drift files a fresh item.
  drift_backlog_ref TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_adoptions_component_id
  ON adoption_records(component_id);

CREATE INDEX IF NOT EXISTS idx_adoptions_scenario
  ON adoption_records(scenario);

CREATE INDEX IF NOT EXISTS idx_adoptions_created_at
  ON adoption_records(created_at DESC);

CREATE TABLE IF NOT EXISTS adoption_files (
  adoption_id              TEXT NOT NULL,
  library_path             TEXT NOT NULL,
  adopted_path             TEXT NOT NULL,
  source_sha256            TEXT NOT NULL DEFAULT '',
  adopted_snapshot_sha256  TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (adoption_id, adopted_path)
);

CREATE INDEX IF NOT EXISTS idx_adoption_files_adoption
  ON adoption_files(adoption_id, library_path);
