-- Notes table — owned by internal/notes/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot (api/main.go through the
-- modules.AllSchemas registry). Times are stored as RFC3339 strings
-- matching the wire format and the time.Time round-trip in
-- sqlite.go::scanNote. Use CREATE TABLE IF NOT EXISTS so re-runs are
-- no-ops.
CREATE TABLE IF NOT EXISTS notes (
  id         TEXT PRIMARY KEY,
  title      TEXT NOT NULL,
  body       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at DESC);

CREATE TABLE IF NOT EXISTS attachments (
  key         TEXT PRIMARY KEY,
  note_id     TEXT NOT NULL,
  mime_type   TEXT NOT NULL DEFAULT '',
  size_bytes  INTEGER NOT NULL,
  uploaded_at TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_attachments_note_uploaded
  ON attachments(note_id, uploaded_at DESC, key DESC);
