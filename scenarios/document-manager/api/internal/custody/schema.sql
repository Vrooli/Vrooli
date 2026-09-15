CREATE TABLE IF NOT EXISTS custody_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_hash TEXT NOT NULL,
  step TEXT NOT NULL,
  tier INTEGER NOT NULL,
  provider TEXT NOT NULL,
  locality TEXT NOT NULL,
  profile TEXT NOT NULL,
  privacy_class TEXT NOT NULL,
  state TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  remedy TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  duration_ms INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_custody_document ON custody_records(document_hash, id);
