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
