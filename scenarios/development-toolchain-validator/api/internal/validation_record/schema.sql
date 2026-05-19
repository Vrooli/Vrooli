-- validation_records table — owned by internal/validation_record/.
-- Append-only history of every terminal validation run. Diff blob
-- never lives here (workspace-sandbox owns it); only diff_hash +
-- diff_path_count for traceability.
CREATE TABLE IF NOT EXISTS validation_records (
  id                                TEXT PRIMARY KEY,
  tuple_kind                        INTEGER NOT NULL,
  subject_id                        TEXT NOT NULL,
  golden_slug                       TEXT NOT NULL,
  started_at                        TEXT NOT NULL,
  ended_at                          TEXT NOT NULL,
  duration_ms                       INTEGER NOT NULL,
  tokens_used                       INTEGER NOT NULL,
  cost_usd_micro                    INTEGER NOT NULL,
  verdict                           INTEGER NOT NULL,
  diff_hash                         TEXT NOT NULL DEFAULT '',
  diff_path_count                   INTEGER NOT NULL DEFAULT 0,
  agent_manager_run_id              TEXT NOT NULL DEFAULT '',
  manifest_template_version_at_run  TEXT NOT NULL DEFAULT '',
  manifest_skill_version_at_run     TEXT NOT NULL DEFAULT '',
  error_message                     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_validation_records_golden     ON validation_records(golden_slug);
CREATE INDEX IF NOT EXISTS idx_validation_records_subject    ON validation_records(subject_id);
CREATE INDEX IF NOT EXISTS idx_validation_records_tuple_kind ON validation_records(tuple_kind);
CREATE INDEX IF NOT EXISTS idx_validation_records_ended_at   ON validation_records(ended_at);
