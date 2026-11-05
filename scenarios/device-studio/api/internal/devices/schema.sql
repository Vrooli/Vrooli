CREATE TABLE IF NOT EXISTS device_revisions (
  device_id TEXT NOT NULL,
  revision INTEGER NOT NULL CHECK (revision > 0),
  sha256 TEXT NOT NULL CHECK (length(sha256) = 64),
  candidate_json TEXT NOT NULL CHECK (length(candidate_json) BETWEEN 2 AND 1048576),
  created_at TEXT NOT NULL,
  PRIMARY KEY (device_id, revision)
);

CREATE INDEX IF NOT EXISTS device_revisions_digest
  ON device_revisions(device_id, sha256, revision DESC);

CREATE TRIGGER IF NOT EXISTS device_revisions_no_update
BEFORE UPDATE ON device_revisions
BEGIN
  SELECT RAISE(ABORT, 'device revisions are immutable');
END;

CREATE TRIGGER IF NOT EXISTS device_revisions_no_delete
BEFORE DELETE ON device_revisions
BEGIN
  SELECT RAISE(ABORT, 'device revisions are immutable');
END;
