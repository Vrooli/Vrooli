CREATE TABLE IF NOT EXISTS disposition (
  signal_id TEXT PRIMARY KEY,
  state TEXT NOT NULL,
  revisit_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS annotation (
  id TEXT PRIMARY KEY,
  signal_id TEXT NOT NULL,
  author TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  outcome_kind TEXT NOT NULL DEFAULT '',
  outcome_target_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS annotation_signal_created_idx ON annotation(signal_id, created_at, id);
