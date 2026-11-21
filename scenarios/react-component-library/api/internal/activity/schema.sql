-- Activity domain schema — owned by internal/activity/.
-- Applied at boot through database.EnsureSchemas via the modules registry.
--
-- activity_events is append-only: the table has no update or delete path in the
-- domain package. before_json/after_json are opaque snapshots supplied by the
-- mutating service so the log can explain a change without knowing its shape.
CREATE TABLE IF NOT EXISTS activity_events (
  id           TEXT PRIMARY KEY,
  actor        TEXT NOT NULL,
  action       TEXT NOT NULL,
  subject_kind TEXT NOT NULL,
  subject_id   TEXT NOT NULL,
  summary      TEXT NOT NULL DEFAULT '',
  before_json  TEXT NOT NULL DEFAULT '',
  after_json   TEXT NOT NULL DEFAULT '',
  at           TEXT NOT NULL,
  trace_id     TEXT NOT NULL DEFAULT ''
);

-- The event log is read newest-first and filtered by subject for detail panes.
CREATE INDEX IF NOT EXISTS idx_activity_events_at
  ON activity_events(at DESC);

CREATE INDEX IF NOT EXISTS idx_activity_events_subject
  ON activity_events(subject_kind, subject_id, at DESC);
