-- gaps — owned by internal/focus/. The persistent gaps registry: the team's
-- accumulated qualitative context (notes/approaches/follow-ups) keyed by gap id,
-- plus registry-only cross-cutting/global gaps. Derived gaps (the non-NOW
-- denominator cells) are joined in live by the service and are NOT stored here;
-- this table only persists what the team adds. Embedded by schema.go and applied
-- via database.EnsureSchemas at boot through the modules.AllSchemas registry.
-- Timestamps are RFC3339Nano (see internal/focus/sqlite.go). notes/approaches/
-- follow_ups are JSON arrays. Use CREATE TABLE IF NOT EXISTS so re-runs are
-- no-ops.
CREATE TABLE IF NOT EXISTS gaps (
  id             TEXT PRIMARY KEY,
  projection     TEXT NOT NULL DEFAULT '',
  title          TEXT NOT NULL DEFAULT '',
  status         TEXT NOT NULL DEFAULT '',
  source_cell_id TEXT NOT NULL DEFAULT '',
  global         INTEGER NOT NULL DEFAULT 0,
  notes          TEXT NOT NULL DEFAULT '[]',
  approaches     TEXT NOT NULL DEFAULT '[]',
  follow_ups     TEXT NOT NULL DEFAULT '[]',
  created_at     TEXT NOT NULL,
  updated_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gaps_projection ON gaps(projection);
