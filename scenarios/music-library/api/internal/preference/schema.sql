CREATE TABLE IF NOT EXISTS music_taste_profile_versions (
  context TEXT NOT NULL,
  revision INTEGER NOT NULL,
  profile_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (context, revision)
);

CREATE TABLE IF NOT EXISTS music_taste_judgments (
  judgment_id TEXT PRIMARY KEY,
  context TEXT NOT NULL,
  left_track_sha256 TEXT NOT NULL,
  right_track_sha256 TEXT NOT NULL,
  choice TEXT NOT NULL CHECK (choice IN ('left','right','tie','neither','skip')),
  judgment_json TEXT NOT NULL,
  judged_at TEXT NOT NULL,
  CHECK (length(left_track_sha256) = 64),
  CHECK (length(right_track_sha256) = 64),
  CHECK (left_track_sha256 <> right_track_sha256)
);
CREATE INDEX IF NOT EXISTS music_taste_judgments_context
  ON music_taste_judgments(context, judged_at, judgment_id);

CREATE TABLE IF NOT EXISTS music_taste_audit_events (
  event_id TEXT PRIMARY KEY,
  context TEXT NOT NULL,
  event_type TEXT NOT NULL,
  event_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS music_taste_audit_context
  ON music_taste_audit_events(context, created_at, event_id);
