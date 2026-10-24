CREATE TABLE IF NOT EXISTS derivation_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_hash TEXT NOT NULL,
  version INTEGER NOT NULL,
  tier INTEGER NOT NULL,
  chain_json TEXT NOT NULL,
  handlers_json TEXT NOT NULL,
  model_json TEXT NOT NULL,
  terminal_state INTEGER NOT NULL,
  reason TEXT NOT NULL,
  remedy TEXT NOT NULL,
  skipped_json TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(document_hash, version)
);

CREATE INDEX IF NOT EXISTS idx_derivation_versions_document ON derivation_versions(document_hash, version);
