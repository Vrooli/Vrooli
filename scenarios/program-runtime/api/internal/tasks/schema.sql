CREATE TABLE IF NOT EXISTS learning_tasks (
  attempt_id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  attempt_number INTEGER NOT NULL,
  session_id TEXT NOT NULL,
  program_id TEXT NOT NULL,
  state TEXT NOT NULL CHECK(state IN ('prepared','running','completed','uncertain')),
  delivery TEXT NOT NULL CHECK(delivery IN ('waiting','pending','delivered','blocked')),
  next_attempt_at TEXT NOT NULL,
  record TEXT NOT NULL,
  parent_attempt_id TEXT,
  step_name TEXT,
  step_path TEXT NOT NULL DEFAULT '',
  UNIQUE(task_id, attempt_number, step_path)
);
CREATE INDEX IF NOT EXISTS learning_tasks_pending ON learning_tasks(delivery,next_attempt_at);
CREATE INDEX IF NOT EXISTS learning_tasks_program ON learning_tasks(program_id);
CREATE TABLE IF NOT EXISTS learning_fragments (
  step_key TEXT NOT NULL,
  fragment_hash TEXT NOT NULL,
  fragment TEXT NOT NULL,
  verified INTEGER NOT NULL DEFAULT 0,
  cached_runs INTEGER NOT NULL DEFAULT 0,
  contradicted_since_edit INTEGER NOT NULL DEFAULT 0,
  source_program_id TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  step_name TEXT NOT NULL DEFAULT '',
  trace_inputs TEXT NOT NULL DEFAULT '{}',
  trace_output TEXT NOT NULL DEFAULT '{}',
  last_used_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY(step_key, fragment_hash)
);
CREATE INDEX IF NOT EXISTS learning_fragments_promotable ON learning_fragments(verified, contradicted_since_edit, last_used_at);

CREATE TABLE IF NOT EXISTS learning_fragment_evidence (
  step_key TEXT NOT NULL,
  fragment_hash TEXT NOT NULL,
  attempt_id TEXT NOT NULL,
  outcome TEXT NOT NULL CHECK(outcome IN ('verified_success','failed')),
  input_digest TEXT NOT NULL DEFAULT '',
  compatibility TEXT NOT NULL DEFAULT '{}',
  evidence TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  PRIMARY KEY(step_key,fragment_hash,attempt_id,outcome)
);

CREATE TABLE IF NOT EXISTS learning_results (
  feedback_ref TEXT PRIMARY KEY,
  attempt_id TEXT NOT NULL,
  provenance TEXT NOT NULL,
  record TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS learning_feedback (
  observation_id TEXT PRIMARY KEY,
  feedback_ref TEXT NOT NULL,
  record TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS learning_findings (
  finding_id TEXT PRIMARY KEY,
  owner TEXT NOT NULL,
  state TEXT NOT NULL,
  record TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS learning_findings_owner_state ON learning_findings(owner,state);
CREATE TABLE IF NOT EXISTS learning_feedback_delivery (
 observation_id TEXT PRIMARY KEY,
 state TEXT NOT NULL DEFAULT 'pending',
 attempts INTEGER NOT NULL DEFAULT 0,
 next_attempt_at TEXT NOT NULL,
 last_error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS learning_feedback_target ON learning_feedback(feedback_ref,created_at);
CREATE INDEX IF NOT EXISTS learning_results_artifact ON learning_results(json_extract(record,'$.artifact'),provenance);
CREATE TABLE IF NOT EXISTS learning_fragment_cohorts (
 step_key TEXT PRIMARY KEY,
 cohort TEXT NOT NULL
);
