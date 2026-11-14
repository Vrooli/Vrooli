CREATE TABLE IF NOT EXISTS operator_session_enrollments (
  reference TEXT PRIMARY KEY,
  operator_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  public_key BLOB NOT NULL,
  scopes_json TEXT NOT NULL DEFAULT '[]',
  enrolled_at TEXT NOT NULL,
  revoked INTEGER NOT NULL DEFAULT 0
);
