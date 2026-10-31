-- Query telemetry tables. Embedded by schema.go and applied through the
-- modules.AllSchemas registry. Forward-only declarative: re-runs are no-ops.
--
-- The router records only hashed query identifiers and aggregate routing facts;
-- corpus content and vectors are owned by providers, not this domain.

CREATE TABLE IF NOT EXISTS query_telemetry (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  query_hash   TEXT NOT NULL,
  routed_types TEXT NOT NULL DEFAULT '',
  result_count INTEGER NOT NULL DEFAULT 0,
  zero_result  INTEGER NOT NULL DEFAULT 0,
  degraded     INTEGER NOT NULL DEFAULT 0,
  reranked     INTEGER NOT NULL DEFAULT 0,
  auto_routed_external INTEGER NOT NULL DEFAULT 0,
  escalated    INTEGER NOT NULL DEFAULT 0,
  latency_ms   INTEGER NOT NULL DEFAULT 0,
  routing_mode TEXT NOT NULL DEFAULT '',
  eligible_provider_count INTEGER NOT NULL DEFAULT 0,
  selected_provider_count INTEGER NOT NULL DEFAULT 0,
  selected_leaf_count INTEGER NOT NULL DEFAULT 0,
  widened_leaf_count INTEGER NOT NULL DEFAULT 0,
  fanout_width_bound_reached INTEGER NOT NULL DEFAULT 0,
  withheld_external_count INTEGER NOT NULL DEFAULT 0,
  queued_provider_count INTEGER NOT NULL DEFAULT 0,
  classifier_latency_ms INTEGER NOT NULL DEFAULT 0,
  resolver_latency_ms INTEGER NOT NULL DEFAULT 0,
  resolver_cache_hits INTEGER NOT NULL DEFAULT 0,
  resolver_cache_misses INTEGER NOT NULL DEFAULT 0,
  fanout_latency_ms INTEGER NOT NULL DEFAULT 0,
  rerank_latency_ms INTEGER NOT NULL DEFAULT 0,
  rerank_candidate_count INTEGER NOT NULL DEFAULT 0,
  response_degrade_reason TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_query_telemetry_created ON query_telemetry(created_at);

CREATE TABLE IF NOT EXISTS query_telemetry_provider (
  query_id       INTEGER NOT NULL REFERENCES query_telemetry(id) ON DELETE CASCADE,
  provider_id    TEXT NOT NULL,
  hit_count      INTEGER NOT NULL DEFAULT 0,
  latency_ms     INTEGER NOT NULL DEFAULT 0,
  degraded       INTEGER NOT NULL DEFAULT 0,
  degrade_reason TEXT NOT NULL DEFAULT '',
  reranker_leg   TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (query_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_query_telemetry_provider_pid ON query_telemetry_provider(provider_id);
