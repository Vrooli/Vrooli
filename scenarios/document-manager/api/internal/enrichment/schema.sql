CREATE TABLE IF NOT EXISTS enrichments (
  document_hash TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  suggested_privacy_class INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS embeddings (
  id TEXT PRIMARY KEY,
  document_hash TEXT NOT NULL,
  unit_id TEXT NOT NULL,
  role TEXT NOT NULL,
  model TEXT NOT NULL,
  dimension INTEGER NOT NULL,
  content_version INTEGER NOT NULL,
  retarget_strategy TEXT NOT NULL,
  vector BLOB NOT NULL,
  created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_embeddings_target ON embeddings(document_hash, unit_id, role, model, content_version);
