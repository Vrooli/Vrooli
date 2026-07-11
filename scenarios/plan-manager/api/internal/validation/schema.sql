-- validation_results — the last-known validation/baseline outcome per plan/phase,
-- owned by internal/validation/. Written by RunValidation (the explicit,
-- agent-in-the-loop check that actually shells the baseline commands) and READ by
-- the execution domain's just-in-time context (status/next) so those poll-style
-- verbs inject the LAST STORED result instead of shelling a subprocess on every
-- call. The verdict/staleness/commands/detail round-trip; commands_run is a JSON
-- array. Rows accumulate (history); the latest by ran_at is the "last" result.
--
-- Embedded by schema.go and applied idempotently via database.EnsureSchemas at
-- boot through the modules.AllSchemas registry. Timestamps are RFC3339Nano. Use
-- CREATE ... IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS validation_results (
  id           TEXT PRIMARY KEY,
  plan_id      TEXT NOT NULL,
  phase_id     TEXT NOT NULL DEFAULT '',
  verdict      TEXT NOT NULL DEFAULT '',
  staleness    TEXT NOT NULL DEFAULT '',
  commands_run TEXT NOT NULL DEFAULT '[]',
  detail       TEXT NOT NULL DEFAULT '',
  ran_at       TEXT NOT NULL
);

-- Lookup is "the latest result for this (plan, phase)" — index the scope + time.
CREATE INDEX IF NOT EXISTS idx_validation_results_scope ON validation_results(plan_id, phase_id, ran_at);

-- validation_operations is the durable ownership record for a long validation.
-- payload_json contains the complete versioned domain record, including child
-- checkpoints and the terminal result. Indexed columns support dedup/recovery
-- without interpreting the payload in SQL.
CREATE TABLE IF NOT EXISTS validation_operations (
  id               TEXT PRIMARY KEY,
  plan_id          TEXT NOT NULL,
  phase_id         TEXT NOT NULL DEFAULT '',
  idempotency_key  TEXT NOT NULL DEFAULT '',
  status           TEXT NOT NULL,
  queued_at        TEXT NOT NULL,
  updated_at       TEXT NOT NULL,
  payload_json     TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_validation_operations_idempotency
ON validation_operations(plan_id, phase_id, idempotency_key)
WHERE idempotency_key <> '';

-- The caller key is only a retry alias. The durable safety boundary is one
-- active graph per plan/phase, including unkeyed starts. Terminal history is
-- intentionally unconstrained so a later validation can be created.
CREATE UNIQUE INDEX IF NOT EXISTS idx_validation_operations_one_active_scope
ON validation_operations(plan_id, phase_id)
WHERE status <> 'terminal';

CREATE INDEX IF NOT EXISTS idx_validation_operations_recovery
ON validation_operations(status, updated_at);
