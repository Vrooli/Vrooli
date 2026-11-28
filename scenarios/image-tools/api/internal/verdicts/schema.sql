-- Operator ratings of job outputs — owned by internal/verdicts/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry.
--
-- One row per rated job (re-rating replaces it). model_id and operation are
-- copied from the job's durable trace when the rating is written, so a verdict
-- stays derivable after the job row is pruned and never has to join a table
-- this domain does not own. Forward-only declarative.
CREATE TABLE IF NOT EXISTS job_rating (
  job_id    TEXT PRIMARY KEY,
  score     INTEGER NOT NULL CHECK (score BETWEEN 1 AND 5),
  note      TEXT NOT NULL DEFAULT '',
  rated_by  TEXT NOT NULL DEFAULT '',
  rated_at  TEXT NOT NULL,
  model_id  TEXT NOT NULL DEFAULT '',
  operation TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_job_rating_model_op ON job_rating (model_id, operation);
