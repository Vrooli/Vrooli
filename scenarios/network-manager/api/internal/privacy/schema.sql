-- Privacy domain — owned by internal/privacy/. Embedded by schema.go and
-- applied through modules.AllSchemas at boot.

CREATE TABLE IF NOT EXISTS retention_settings (
  id              TEXT PRIMARY KEY,
  query_log_days  INTEGER NOT NULL,
  snapshot_days   INTEGER NOT NULL,
  experiment_days INTEGER NOT NULL,
  profile         TEXT NOT NULL,
  updated_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS visibility_settings (
  id                  TEXT PRIMARY KEY,
  show_query_domains  INTEGER NOT NULL,
  show_device_history INTEGER NOT NULL,
  household_mode      INTEGER NOT NULL,
  notes_json          TEXT NOT NULL DEFAULT '[]',
  updated_at          TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS privacy_sweep_records (
  id                TEXT PRIMARY KEY,
  profile           TEXT NOT NULL,
  snapshot_cutoff   TEXT NOT NULL DEFAULT '',
  snapshots_deleted INTEGER NOT NULL DEFAULT 0,
  notes_json        TEXT NOT NULL DEFAULT '[]',
  created_at        TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_privacy_sweep_records_created
  ON privacy_sweep_records(created_at DESC, id DESC);
