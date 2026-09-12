-- Operation runtime measures (IMG-P0-012) — owned by internal/measures/.
-- Embedded by schema.go and applied via database.EnsureSchemas at boot through
-- the modules.AllSchemas registry.
--
-- One row per finalized operation execution. Op-level facts (latency, queue
-- wait, terminal state) are recorded for every job via the job Manager's
-- OnComplete hook; model_id/tier/fallback are filled when the AI runner records
-- a model-backed sample. Aggregates (p50/p95 latency, throughput, fallback mix)
-- are computed on read. Forward-only declarative.
CREATE TABLE IF NOT EXISTS op_measure (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  operation     TEXT NOT NULL,
  model_id      TEXT NOT NULL DEFAULT '',
  tier          TEXT NOT NULL DEFAULT '',
  state         TEXT NOT NULL DEFAULT '',
  duration_ms   INTEGER NOT NULL DEFAULT 0,
  queue_wait_ms INTEGER NOT NULL DEFAULT 0,
  fallback_used INTEGER NOT NULL DEFAULT 0,
  recorded_at   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_op_measure_operation ON op_measure (operation);

-- Structured terminal job traces (Phase 5 observability). This table records
-- one row per finalized job with enough context to answer "what ran, where, and
-- how long did it wait/run?" without scraping logs. Additive table, separate
-- from jobs so existing databases do not need an ALTER migration.
CREATE TABLE IF NOT EXISTS job_trace (
  job_id       TEXT PRIMARY KEY,
  operation    TEXT NOT NULL,
  model_id     TEXT NOT NULL DEFAULT '',
  backend      TEXT NOT NULL DEFAULT '',
  tier         TEXT NOT NULL DEFAULT '',
  lane         TEXT NOT NULL DEFAULT '',
  state        TEXT NOT NULL DEFAULT '',
  duration_ms  INTEGER NOT NULL DEFAULT 0,
  queue_wait_ms INTEGER NOT NULL DEFAULT 0,
  result_ref   TEXT NOT NULL DEFAULT '',
  error        TEXT NOT NULL DEFAULT '',
  recorded_at  TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_job_trace_operation ON job_trace (operation);
CREATE INDEX IF NOT EXISTS idx_job_trace_model_id ON job_trace (model_id);
