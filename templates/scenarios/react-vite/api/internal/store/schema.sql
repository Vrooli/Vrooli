-- SQLite schema for {{SCENARIO_ID}}. Embedded at build time by schema.go
-- and applied on every API startup via store.EnsureSchema. Use
-- CREATE TABLE IF NOT EXISTS so the script is idempotent across restarts.

-- notes is the canonical CRUD reference table. Backs the /api/v1/notes
-- endpoints exposed in handlers/notes/. Times are stored as RFC3339
-- strings (matching the wire format and the time.Time round-trip in
-- internal/notes/sqlite.go::scanNote).
CREATE TABLE IF NOT EXISTS notes (
  id         TEXT PRIMARY KEY,
  title      TEXT NOT NULL,
  body       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at DESC);
