CREATE TABLE IF NOT EXISTS motion_revisions (
  motion_id TEXT NOT NULL,
  revision INTEGER NOT NULL CHECK (revision > 0),
  sha256 TEXT NOT NULL CHECK (length(sha256) = 64),
  candidate_json TEXT NOT NULL CHECK (length(candidate_json) BETWEEN 2 AND 1048576),
  created_at TEXT NOT NULL,
  PRIMARY KEY (motion_id, revision)
);

CREATE TRIGGER IF NOT EXISTS motion_revisions_no_update
BEFORE UPDATE ON motion_revisions
BEGIN SELECT RAISE(ABORT, 'motion revisions are immutable'); END;

CREATE TRIGGER IF NOT EXISTS motion_revisions_no_delete
BEFORE DELETE ON motion_revisions
BEGIN SELECT RAISE(ABORT, 'motion revisions are immutable'); END;
