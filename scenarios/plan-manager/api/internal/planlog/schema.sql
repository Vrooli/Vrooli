-- log_entries — owned by internal/planlog/.
-- Plan Manager's single execution-log ledger (docs/concepts/DATA.md): typed
-- entries for decisions, candidate findings, filed bug reports, reusable
-- records, and notes. Findings, bug reports, and records are DISTINCT types in
-- one table, scoped to a plan/execution/phase and carrying downstream sync state
-- so a failed forward stays durable and retryable.
--
-- Embedded by schema.go and applied idempotently via database.EnsureSchemas at
-- boot through the modules.AllSchemas registry. Timestamps are RFC3339Nano. Use
-- CREATE ... IF NOT EXISTS so re-runs are no-ops. Pool=1 SQLite: callers collect
-- rows into slices before any follow-up query.

CREATE TABLE IF NOT EXISTS log_entries (
  id                 TEXT PRIMARY KEY,
  type               TEXT NOT NULL,
  plan_id            TEXT NOT NULL DEFAULT '',
  execution_id       TEXT NOT NULL DEFAULT '',
  phase_id           TEXT NOT NULL DEFAULT '',
  title              TEXT NOT NULL DEFAULT '',
  detail             TEXT NOT NULL DEFAULT '',
  severity           TEXT NOT NULL DEFAULT '',
  triage             TEXT NOT NULL DEFAULT '',
  sync_status        TEXT NOT NULL DEFAULT 'local',
  downstream         TEXT NOT NULL DEFAULT '{}',
  bug_payload        TEXT NOT NULL DEFAULT '{}',
  record_payload     TEXT NOT NULL DEFAULT '{}',
  capture            TEXT NOT NULL DEFAULT '{}',
  source_command     TEXT NOT NULL DEFAULT '',
  evidence           TEXT NOT NULL DEFAULT '[]',
  attribution_run_id TEXT NOT NULL DEFAULT '',
  idempotency_key    TEXT NOT NULL DEFAULT '',
  supersedes_id      TEXT NOT NULL DEFAULT '',
  promoted_from_id   TEXT NOT NULL DEFAULT '',
  created_at         TEXT NOT NULL,
  updated_at         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_log_entries_plan ON log_entries(plan_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_execution ON log_entries(execution_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_type ON log_entries(type);
CREATE INDEX IF NOT EXISTS idx_log_entries_sync ON log_entries(sync_status);

-- Idempotency key dedup, authoritative at the store: a retry that carries the
-- same (plan_id, idempotency_key) cannot create a second entry. PARTIAL (only
-- rows that carry a key) so keyless entries still insert freely.
CREATE UNIQUE INDEX IF NOT EXISTS idx_log_entries_idempotency
  ON log_entries(plan_id, idempotency_key)
  WHERE idempotency_key <> '';

-- Attribution-keyed dedup for findings/decisions without an explicit idempotency
-- key: the same (plan_id, execution_id, attribution_run_id, type, normalized
-- title) is not double-filed. plan_id is included so entries attached to a plan
-- handle (empty execution_id) stay scoped to their plan and never collide across
-- plans. PARTIAL on attribution_run_id presence.
CREATE UNIQUE INDEX IF NOT EXISTS idx_log_entries_attribution_dedup
  ON log_entries(plan_id, execution_id, attribution_run_id, type, lower(trim(title)))
  WHERE attribution_run_id <> '';
