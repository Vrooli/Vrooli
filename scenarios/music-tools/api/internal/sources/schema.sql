CREATE TABLE IF NOT EXISTS music_sources (
  sha256 TEXT PRIMARY KEY,
  blob_key TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  artist TEXT NOT NULL DEFAULT '',
  uploader TEXT NOT NULL DEFAULT '',
  source_url TEXT NOT NULL DEFAULT '',
  acquired_at TEXT NOT NULL DEFAULT '',
  file_name TEXT NOT NULL,
  media_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  CHECK (length(sha256) = 64),
  CHECK (size_bytes > 0 AND size_bytes <= 134217728)
);
CREATE INDEX IF NOT EXISTS music_sources_created_at ON music_sources(created_at DESC, sha256);
