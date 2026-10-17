-- Corpus tables — owned by internal/corpus/. Embedded by schema.go and applied
-- via database.EnsureSchemas at boot (api/main.go through the modules.AllSchemas
-- registry). Holds ONLY metadata: the audio bytes live in the blob store under
-- the git-ignored runtime data dir, addressed by blob_key. tags is a JSON text
-- array; created_at is an RFC3339Nano string matching the time.Time round-trip
-- in sqlite.go. Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS corpus_clips (
  id             TEXT PRIMARY KEY,
  reference_text TEXT NOT NULL DEFAULT '',
  tags           TEXT NOT NULL DEFAULT '[]',
  duration_ms    INTEGER NOT NULL DEFAULT 0,
  sample_rate_hz INTEGER NOT NULL DEFAULT 0,
  format         TEXT NOT NULL DEFAULT '',
  blob_key       TEXT NOT NULL,
  source         TEXT NOT NULL DEFAULT 'free_form',
  created_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_corpus_clips_created_at ON corpus_clips(created_at DESC);
