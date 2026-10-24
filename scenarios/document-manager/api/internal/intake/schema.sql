CREATE TABLE IF NOT EXISTS documents (
  id TEXT PRIMARY KEY,
  content_sha256 TEXT NOT NULL UNIQUE,
  source_name TEXT NOT NULL DEFAULT '',
  detected_mime TEXT NOT NULL,
  pdf_type TEXT NOT NULL DEFAULT '',
  pdf_confidence REAL NOT NULL DEFAULT 0,
  privacy_class TEXT NOT NULL DEFAULT 'internal',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_documents_source_name ON documents(source_name);
