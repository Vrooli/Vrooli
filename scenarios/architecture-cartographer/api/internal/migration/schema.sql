-- Migration tables — owned by internal/migration/. Persist the stateful
-- tracker: one row per migration, one row per tracked finding. Findings
-- are source-agnostic (the shared ArchitectureFinding contract); the
-- reconciliation key is the afid stable_id.
--
-- A finding's identity inside a migration is (migration_id, stable_id):
-- re-audits match purely on stable_id, so the same defect collapses onto
-- the same row across audits and the lifecycle state survives.

CREATE TABLE IF NOT EXISTS migrations (
  id          TEXT PRIMARY KEY,
  scenario    TEXT NOT NULL,
  name        TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_migrations_scenario
  ON migrations(scenario, created_at DESC);

CREATE TABLE IF NOT EXISTS migration_findings (
  migration_id    TEXT NOT NULL,
  stable_id       TEXT NOT NULL,
  scenario        TEXT NOT NULL,
  source          TEXT NOT NULL,
  code            TEXT NOT NULL,
  severity        TEXT NOT NULL,
  locations       TEXT NOT NULL DEFAULT '[]',
  domains         TEXT NOT NULL DEFAULT '[]',
  message         TEXT NOT NULL DEFAULT '',
  suggestion      TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL,
  resolution_note TEXT NOT NULL DEFAULT '',
  regressed       INTEGER NOT NULL DEFAULT 0,
  first_seen_at   TEXT NOT NULL,
  updated_at      TEXT NOT NULL,
  PRIMARY KEY (migration_id, stable_id),
  FOREIGN KEY (migration_id) REFERENCES migrations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_migration_findings_status
  ON migration_findings(migration_id, status);
