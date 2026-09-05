-- Conflicts tables — owned by internal/conflicts/. Detection-only:
-- persists the detected conflict envelope (the photograph). There is no
-- lifecycle here — the stateful per-finding lifecycle lives in the
-- migration domain (migration_findings). The conflict payload (evidence,
-- suggested_fixes, verdict, locations, domains) lives in a single JSON
-- blob so adding optional fields to Fix does not require a migration; the
-- canonical envelope (id, detector, type, severity, etc.) lives in columns
-- for indexable queries.
--
-- v0.2: `id` is the deterministic content-hash stable_id produced by
-- conflicts.StableID. `instance_id` is the per-run UUID preserved for
-- log correlation. Two runs that detect the same underlying drift
-- collapse onto the same row via ON CONFLICT(id) DO UPDATE.

CREATE TABLE IF NOT EXISTS conflicts (
  id               TEXT PRIMARY KEY,
  instance_id      TEXT NOT NULL DEFAULT '',
  scenario         TEXT NOT NULL,
  detector         TEXT NOT NULL,
  type             TEXT NOT NULL,
  subtype          TEXT NOT NULL DEFAULT '',
  severity         TEXT NOT NULL,
  snapshot_id      TEXT NOT NULL DEFAULT '',
  payload          BLOB,
  detected_at      TEXT NOT NULL,
  updated_at       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_conflicts_scenario
  ON conflicts(scenario, detected_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_conflicts_type
  ON conflicts(scenario, type, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_conflicts_instance
  ON conflicts(instance_id);
