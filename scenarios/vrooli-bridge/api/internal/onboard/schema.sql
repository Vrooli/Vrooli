-- Onboarding ops + step events — owned by internal/onboard/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot (api/main.go through
-- modules.AllSchemas). The phase-5 orchestration tier: durable, server-owned
-- one-shot node onboarding ops and their append-only step-event history. Times
-- are RFC3339Nano strings matching the wire format and the time.Time round-trip
-- in sqlite.go. Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops (migrate,
-- never recreate). Postgres-compatible column types for forward scale.
--
-- SECRETS never land here: there is deliberately NO column for the owner SSH
-- password or the single-use pairing code. The password is request-scoped and
-- zeroed after first touch (installing the SSH key and, when requested, the
-- passwordless-sudo drop-in — which the password only ever reaches over `sudo -S`
-- on stdin, never argv or a log line); the pairing code is issued server-side and
-- injected into the remote bootstrap over stdin. The only durable identity is
-- node_id, learned once the bootstrap redeems the code.
CREATE TABLE IF NOT EXISTS onboarding_ops (
  id              TEXT PRIMARY KEY,
  host            TEXT NOT NULL DEFAULT '',
  port            INTEGER NOT NULL DEFAULT 0,
  user_name       TEXT NOT NULL DEFAULT '',
  node_name       TEXT NOT NULL DEFAULT '',
  target_revision TEXT NOT NULL DEFAULT '',
  repo_url        TEXT NOT NULL DEFAULT '',
  state           INTEGER NOT NULL DEFAULT 0,
  node_id         TEXT NOT NULL DEFAULT '',
  failure_reason  TEXT NOT NULL DEFAULT '',
  -- A bounded, multi-line tail of the node-side diagnostic output on a FAILED op
  -- (the concrete cause behind failure_reason — e.g. the `make setup` error).
  -- Never secret material. A DB created before this column is brought to shape by
  -- Migrate() (guarded ALTER TABLE ADD COLUMN, run before EnsureSchemas).
  failure_detail  TEXT NOT NULL DEFAULT '',
  exit_code       INTEGER NOT NULL DEFAULT 0,
  created_at      TEXT NOT NULL,
  started_at      TEXT NOT NULL DEFAULT '',
  finished_at     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_onboarding_ops_created_at ON onboarding_ops(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_onboarding_ops_host ON onboarding_ops(host);

-- onboarding_step_events is append-only: rows are only ever INSERTed and
-- SELECTed (there is no UPDATE/DELETE in sqlite.go). The (op_id, sequence) pair
-- is unique so a replayed append is de-duplicated rather than double-stored.
CREATE TABLE IF NOT EXISTS onboarding_step_events (
  op_id      TEXT NOT NULL,
  sequence   INTEGER NOT NULL,
  step_id    TEXT NOT NULL DEFAULT '',
  status     INTEGER NOT NULL DEFAULT 0,
  detail     TEXT NOT NULL DEFAULT '',
  emitted_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (op_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_onboarding_step_events_op ON onboarding_step_events(op_id, sequence);
