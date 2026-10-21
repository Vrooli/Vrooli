-- Durable Device Studio render job domain. Binary media and checkpoints are
-- owned by blob/artifact storage; these rows retain immutable refs and receipts.
CREATE TABLE IF NOT EXISTS render_jobs (
  id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  state TEXT NOT NULL CHECK (state IN ('queued','running','checkpointed','cancel_requested','succeeded','failed','cancelled')),
  version INTEGER NOT NULL CHECK (version > 0),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  record_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_render_jobs_state_updated ON render_jobs(state, updated_at);
CREATE TABLE IF NOT EXISTS render_job_attempts (
  id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL REFERENCES render_jobs(id) ON DELETE CASCADE,
  attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
  state TEXT NOT NULL CHECK (state IN ('queued','running','checkpointed','succeeded','failed','cancelled')),
  record_json TEXT NOT NULL,
  UNIQUE(job_id, attempt_number)
);
CREATE INDEX IF NOT EXISTS idx_render_job_attempts_job_number ON render_job_attempts(job_id, attempt_number);
-- A claimed native execution remains discoverable after a terminal job state
-- until its exact owner unit has been released. Receipts are durable audit
-- records; the pending table is the bounded lifecycle pump's restart index.
CREATE TABLE IF NOT EXISTS render_job_release_pending (
  job_id TEXT NOT NULL REFERENCES render_jobs(id) ON DELETE CASCADE,
  attempt_id TEXT NOT NULL REFERENCES render_job_attempts(id) ON DELETE CASCADE,
  PRIMARY KEY(job_id, attempt_id)
);
CREATE INDEX IF NOT EXISTS idx_render_job_release_pending_job ON render_job_release_pending(job_id, attempt_id);
CREATE TABLE IF NOT EXISTS render_job_release_receipts (
  job_id TEXT NOT NULL REFERENCES render_jobs(id) ON DELETE CASCADE,
  attempt_id TEXT NOT NULL REFERENCES render_job_attempts(id) ON DELETE CASCADE,
  released_at INTEGER NOT NULL,
  PRIMARY KEY(job_id, attempt_id)
);
-- Typed subjects project known terminal job results into their owning domain.
-- This marker is committed with the terminal transition and retried idempotently.
CREATE TABLE IF NOT EXISTS render_job_subject_finalization_pending (
  job_id TEXT NOT NULL REFERENCES render_jobs(id) ON DELETE CASCADE,
  attempt_id TEXT NOT NULL REFERENCES render_job_attempts(id) ON DELETE CASCADE,
  subject_kind TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY(job_id, attempt_id)
);
CREATE INDEX IF NOT EXISTS idx_render_job_subject_finalization_job ON render_job_subject_finalization_pending(job_id,attempt_id);
