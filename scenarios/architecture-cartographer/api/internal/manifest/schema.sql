-- Manifest cache — owned by internal/manifest/. Parsed manifests are
-- stored as canonical JSON so the proto/Go shape remains the
-- source-of-truth. Lookups are by scenario; content_hash lets the
-- service decide whether to re-validate when the source bytes change.

CREATE TABLE IF NOT EXISTS manifests (
  scenario      TEXT PRIMARY KEY,
  version       TEXT NOT NULL,
  content_hash  TEXT NOT NULL,
  payload       BLOB NOT NULL,
  parsed_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_manifests_content_hash
  ON manifests(content_hash);
