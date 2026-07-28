CREATE TABLE IF NOT EXISTS entries (
  id TEXT PRIMARY KEY,
  body TEXT NOT NULL,
  facet_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT '',
  actor_kind TEXT NOT NULL DEFAULT '',
  source_runtime TEXT NOT NULL DEFAULT '',
  run_id TEXT NOT NULL DEFAULT '',
  workflow_execution_id TEXT NOT NULL DEFAULT '',
  import_key TEXT,
  source_harness TEXT NOT NULL DEFAULT '',
  source_path TEXT NOT NULL DEFAULT '',
  imported_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_entries_import_key ON entries(import_key) WHERE import_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_entries_created_at ON entries(created_at ASC, id ASC);
CREATE TABLE IF NOT EXISTS facet_texts (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  text TEXT NOT NULL,
  embedding_ref TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (entry_id) REFERENCES entries(id)
);
CREATE TABLE IF NOT EXISTS embeddings (
  id TEXT PRIMARY KEY,
  facet_text_id TEXT NOT NULL,
  vector_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (facet_text_id) REFERENCES facet_texts(id)
);
CREATE TABLE IF NOT EXISTS journal_retry_queue (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (entry_id) REFERENCES entries(id)
);
