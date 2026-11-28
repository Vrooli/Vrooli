package prose

const schemaSQL = `
CREATE TABLE IF NOT EXISTS prose_records (
  kind TEXT NOT NULL, record_key TEXT NOT NULL, version INTEGER NOT NULL,
  payload TEXT NOT NULL, authority TEXT NOT NULL DEFAULT 'local', source_path TEXT,
  content_hash TEXT, status TEXT NOT NULL DEFAULT 'active', frozen INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL, PRIMARY KEY(kind, record_key, version)
);
CREATE TABLE IF NOT EXISTS prose_sessions (id TEXT PRIMARY KEY, payload TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS prose_rounds (id TEXT PRIMARY KEY, session_id TEXT, payload TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS prose_candidates (id TEXT PRIMARY KEY, round_id TEXT, payload TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS prose_selection_events (id TEXT PRIMARY KEY, session_id TEXT, payload TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS prose_documents (id TEXT PRIMARY KEY, payload TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS prose_sections (id TEXT PRIMARY KEY, document_id TEXT NOT NULL, payload TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS prose_declarations (path TEXT PRIMARY KEY, record_key TEXT NOT NULL, payload TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS prose_records_lookup ON prose_records(kind, record_key, version DESC);
`

func Schema() string { return schemaSQL }
