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
-- Durable parent scopes fence a bounded group of otherwise ordinary jobs.
-- Domain policy stays as a pinned document; the jobs owner atomically counts
-- admissions and checks scope state while claiming work.
CREATE TABLE IF NOT EXISTS render_job_scopes (
  id TEXT PRIMARY KEY,
  owner_kind TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('active','closing','cancelled','completed')),
  policy_json TEXT NOT NULL,
  max_children INTEGER NOT NULL CHECK (max_children > 0),
  max_provider_calls INTEGER NOT NULL CHECK (max_provider_calls >= 0),
  children_admitted INTEGER NOT NULL DEFAULT 0 CHECK (children_admitted >= 0),
  provider_calls_reserved INTEGER NOT NULL DEFAULT 0 CHECK (provider_calls_reserved >= 0),
  version INTEGER NOT NULL CHECK (version > 0),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS render_job_scope_children (
  scope_id TEXT NOT NULL REFERENCES render_job_scopes(id),
  job_id TEXT NOT NULL UNIQUE REFERENCES render_jobs(id) ON DELETE CASCADE,
  input_fingerprint TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  provider_calls INTEGER NOT NULL CHECK (provider_calls >= 0),
  PRIMARY KEY(scope_id, ordinal),
  UNIQUE(scope_id, job_id)
);
CREATE INDEX IF NOT EXISTS idx_render_job_scope_children_scope ON render_job_scope_children(scope_id, ordinal);
