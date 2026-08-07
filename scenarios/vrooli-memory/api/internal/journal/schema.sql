CREATE TABLE IF NOT EXISTS entries (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL DEFAULT 'agent-memory',
  body TEXT NOT NULL,
  facet_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT '',
  actor_kind TEXT NOT NULL DEFAULT '',
  source_runtime TEXT NOT NULL DEFAULT '',
  verification_status TEXT NOT NULL DEFAULT 'absent',
  harness_session_id TEXT NOT NULL DEFAULT '',
  harness_kind TEXT NOT NULL DEFAULT '',
  run_id TEXT NOT NULL DEFAULT '',
  workflow_execution_id TEXT NOT NULL DEFAULT '',
  import_key TEXT,
  source_harness TEXT NOT NULL DEFAULT '',
  source_path TEXT NOT NULL DEFAULT '',
  imported_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_entries_scope_import_key ON entries(scope, import_key) WHERE import_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_entries_created_at ON entries(created_at ASC, id ASC);
CREATE TABLE IF NOT EXISTS journal_high_water_mark (
  id INTEGER PRIMARY KEY CHECK (id=1),
  max_rowid INTEGER NOT NULL,
  recorded_at TEXT NOT NULL
);
INSERT OR IGNORE INTO journal_high_water_mark(id,max_rowid,recorded_at)
SELECT 1,COALESCE(MAX(rowid),0),CURRENT_TIMESTAMP FROM entries;
CREATE TRIGGER IF NOT EXISTS entries_append_only_update
BEFORE UPDATE ON entries
BEGIN
  SELECT RAISE(ABORT, 'journal entries are append-only');
END;
CREATE TRIGGER IF NOT EXISTS entries_append_only_delete
BEFORE DELETE ON entries
BEGIN
  SELECT RAISE(ABORT, 'journal entries are append-only');
END;
CREATE TABLE IF NOT EXISTS facet_texts (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  text TEXT NOT NULL,
  embedding_ref TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (entry_id) REFERENCES entries(id)
);
CREATE INDEX IF NOT EXISTS idx_facet_texts_entry ON facet_texts(entry_id, id);
CREATE TABLE IF NOT EXISTS embeddings (
  id TEXT PRIMARY KEY,
  facet_text_id TEXT NOT NULL,
  vector_json TEXT NOT NULL,
  vector_blob BLOB NOT NULL DEFAULT X'',
  created_at TEXT NOT NULL,
  FOREIGN KEY (facet_text_id) REFERENCES facet_texts(id)
);
CREATE INDEX IF NOT EXISTS idx_embeddings_facet_text ON embeddings(facet_text_id, id);
CREATE TABLE IF NOT EXISTS journal_retry_queue (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (entry_id) REFERENCES entries(id)
);
