-- Conflicts tables — owned by internal/conflicts/. Persists the
-- detected conflict envelope + the lifecycle state. The conflict
-- payload (evidence, suggested_fixes, verdict, locations, domains)
-- lives in a single JSON blob so adding optional fields to Fix does
-- not require a migration; the canonical envelope (id, detector, type,
-- status, etc.) lives in columns for indexable queries.

CREATE TABLE IF NOT EXISTS conflicts (
  id               TEXT PRIMARY KEY,
  scenario         TEXT NOT NULL,
  detector         TEXT NOT NULL,
  type             TEXT NOT NULL,
  subtype          TEXT NOT NULL DEFAULT '',
  severity         TEXT NOT NULL,
  status           TEXT NOT NULL,
  assigned_domain  TEXT NOT NULL DEFAULT '',
  resolution_note  TEXT NOT NULL DEFAULT '',
  snapshot_id      TEXT NOT NULL DEFAULT '',
  payload          BLOB,
  detected_at      TEXT NOT NULL,
  updated_at       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_conflicts_scenario_status
  ON conflicts(scenario, status, detected_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_conflicts_type
  ON conflicts(scenario, type, detected_at DESC);
