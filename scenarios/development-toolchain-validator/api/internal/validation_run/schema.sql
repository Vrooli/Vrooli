-- validation_runs table — owned by internal/validation_run/.
-- Operational state for in-flight and recently-completed runs. The
-- canonical append-only history lives in validation_records. Both
-- exist by design: validation_runs is the "currently running" view
-- and the trail of how a record got to terminal; validation_records
-- is the per-run terminal snapshot used for reporting.
CREATE TABLE IF NOT EXISTS validation_runs (
  id                    TEXT PRIMARY KEY,
  tuple_kind            INTEGER NOT NULL,
  subject_id            TEXT NOT NULL,
  golden_slug           TEXT NOT NULL,
  status                INTEGER NOT NULL,
  terminal_verdict      INTEGER NOT NULL DEFAULT 0,
  agent_manager_run_id  TEXT NOT NULL DEFAULT '',
  created_at            TEXT NOT NULL,
  started_at            TEXT NOT NULL DEFAULT '',
  ended_at              TEXT NOT NULL DEFAULT '',
  error_message         TEXT NOT NULL DEFAULT '',
  force_re_run          INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_validation_runs_status     ON validation_runs(status);
CREATE INDEX IF NOT EXISTS idx_validation_runs_golden     ON validation_runs(golden_slug);
CREATE INDEX IF NOT EXISTS idx_validation_runs_subject    ON validation_runs(subject_id);
CREATE INDEX IF NOT EXISTS idx_validation_runs_created_at ON validation_runs(created_at);
