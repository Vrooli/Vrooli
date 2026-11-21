-- authoring_sessions — owned by internal/authoring/. The transient state of a
-- guided composer flow, persisted so a session survives across CLI invocations
-- (each `author …` command is a separate process).
--
-- A session's first-class queryable columns (id/title/slug/finalized/plan_id/
-- timestamps) sit alongside a `document` JSON column that carries the ordered
-- sections[] plus the current-section pointer. Sections are always loaded with
-- their session and are never queried across sessions, so they persist within
-- the document rather than as a separate table (this shape avoids the SQLite
-- pool=1 nested-query deadlock and keeps round-trips deterministic). The plan a
-- session finalizes into is owned by internal/plans (plan_id is a soft pointer).
--
-- Embedded by schema.go and applied idempotently via database.EnsureSchemas at
-- boot through the modules.AllSchemas registry. Timestamps are RFC3339Nano. Use
-- CREATE ... IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS authoring_sessions (
  id         TEXT PRIMARY KEY,
  title      TEXT NOT NULL DEFAULT '',
  slug       TEXT NOT NULL DEFAULT '',
  finalized  INTEGER NOT NULL DEFAULT 0,
  plan_id    TEXT NOT NULL DEFAULT '',
  document   TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_authoring_sessions_finalized ON authoring_sessions(finalized);
