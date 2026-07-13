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

-- V1 indexes scoped every active ticket to a plan/phase. That incorrectly
-- collapsed a new execution or scope generation (and even a distinct explicit
-- request key) onto stale evidence. The JSON document is versioned durable
-- state, so expression indexes let us evolve this ownership boundary without
-- rewriting historical operation rows or adding a fragile table migration.
DROP INDEX IF EXISTS idx_validation_operations_idempotency;
DROP INDEX IF EXISTS idx_validation_operations_one_active_scope;

-- An explicit key identifies one logical producer request within the guided
-- execution and its scope generation. Replaying that request is safe; a
-- distinct key intentionally creates fresh evidence.
CREATE UNIQUE INDEX IF NOT EXISTS idx_validation_operations_idempotency
ON validation_operations(
  plan_id,
  phase_id,
  COALESCE(json_extract(payload_json, '$.ExecutionID'), ''),
  COALESCE(CAST(json_extract(payload_json, '$.ScopeGeneration') AS INTEGER), 0),
  idempotency_key
)
WHERE idempotency_key <> '';

-- Unkeyed starts have no caller-provided logical identity, so only they
-- coalesce while non-terminal. Keyed starts are already protected by the
-- replay index above and must be free to request fresh producer evidence.
CREATE UNIQUE INDEX IF NOT EXISTS idx_validation_operations_one_active_scope
ON validation_operations(
  plan_id,
  phase_id,
  COALESCE(json_extract(payload_json, '$.ExecutionID'), ''),
  COALESCE(CAST(json_extract(payload_json, '$.ScopeGeneration') AS INTEGER), 0)
)
WHERE status <> 'terminal' AND idempotency_key = '';

CREATE INDEX IF NOT EXISTS idx_validation_operations_recovery
ON validation_operations(status, updated_at);
