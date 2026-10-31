-- Eval table set — owned by internal/eval/. Embedded by schema.go and applied
-- via database.EnsureSchemas at boot (api/main.go through modules.AllSchemas).
-- Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops (forward-only declarative).
--
-- Two tables, mirroring registry/metrics:
--   eval_suites — one row = one provider-owned golden suite. The full EvalSuite
--     is persisted as a protojson blob in `descriptor` (source of truth); the
--     projected columns (provider_id, name, state) exist only so ListSuites can
--     filter without parsing every blob.
--   eval_runs — one row = one immutable execution of a suite. The full EvalRun
--     (config snapshot + per-case results + aggregate) is the protojson blob in
--     `result`; the projected columns (suite_id, tag, created_at) drive the
--     history/trend/compare reads without parsing every blob.
--
-- Times are RFC3339Nano strings, matching the registry convention and the wire
-- format. The hub holds NO corpus content here — only suite definitions and the
-- runs taken against them.
CREATE TABLE IF NOT EXISTS eval_suites (
  suite_id    TEXT PRIMARY KEY,
  provider_id TEXT NOT NULL DEFAULT '',
  name        TEXT NOT NULL DEFAULT '',
  state       TEXT NOT NULL DEFAULT 'active',
  descriptor  TEXT NOT NULL,            -- EvalSuite protojson blob (source of truth)
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS eval_runs (
  run_id      TEXT PRIMARY KEY,
  suite_id    TEXT NOT NULL,
  tag         TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL,
  result      TEXT NOT NULL             -- EvalRun protojson blob (immutable)
);

CREATE INDEX IF NOT EXISTS idx_eval_suites_provider ON eval_suites(provider_id);
CREATE INDEX IF NOT EXISTS idx_eval_runs_suite      ON eval_runs(suite_id);
CREATE INDEX IF NOT EXISTS idx_eval_runs_tag        ON eval_runs(tag);
CREATE INDEX IF NOT EXISTS idx_eval_runs_created    ON eval_runs(created_at);

-- Scheduled three-way label validation is retained separately from immutable
-- eval runs. A provider_error verdict must remain distinguishable from stale
-- evidence after the scheduler process restarts.
CREATE TABLE IF NOT EXISTS eval_corpus_validations (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  suite_id    TEXT NOT NULL,
  created_at  TEXT NOT NULL,
  result      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_eval_validations_suite_created
  ON eval_corpus_validations(suite_id, created_at);
