-- Audit log — owned by internal/audit/. Records security-relevant auth events
-- (sign-in/out, token-family revoke, lockout). Applied via EnsureSchemas at
-- boot. Times are RFC3339Nano strings. Forward-only declarative.
CREATE TABLE IF NOT EXISTS audit_log (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL DEFAULT '',
  realm_id   TEXT NOT NULL DEFAULT '',
  action     TEXT NOT NULL,
  ip_address TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  success    INTEGER NOT NULL DEFAULT 0,
  metadata   TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id, created_at DESC);
