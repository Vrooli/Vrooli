-- Recovery events table — owned by internal/recovery/. The durable log
-- of every auto-recovery attempt (trigger, action, outcome, attempt#).
-- Embedded by schema.go and applied via database.EnsureSchemas at boot.
-- Times are RFC3339Nano strings matching the wire format and the
-- time.Time round-trip in sqlite.go::scanEvent. Use CREATE TABLE IF NOT
-- EXISTS so re-runs are no-ops; add columns with ALTER TABLE ... ADD
-- COLUMN (migrate, never recreate).
CREATE TABLE IF NOT EXISTS recovery_events (
  id         TEXT PRIMARY KEY,
  trigger    TEXT NOT NULL,
  action     TEXT NOT NULL,
  outcome    TEXT NOT NULL,
  details    TEXT NOT NULL DEFAULT '',
  attempt    INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_recovery_events_created_at ON recovery_events(created_at);
