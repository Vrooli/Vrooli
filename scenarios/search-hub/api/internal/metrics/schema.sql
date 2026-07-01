-- Metrics (telemetry) tables — owned by internal/metrics/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot (api/main.go
-- through the modules.AllSchemas registry). Use CREATE TABLE IF NOT EXISTS so
-- re-runs are no-ops (forward-only declarative).
--
-- This is the ONLY persistence the thin router owns beyond the provider
-- registry. It holds NO corpus content and NO vectors — only per-query
-- telemetry. The query text is HASHED (never stored raw) so the table carries
-- no recoverable user input (the plan's privacy note).
--
-- Two tables, one normalized layer:
--   query_telemetry          — one row per federated query (latency + flags +
--                              total result count).
--   query_telemetry_provider — one row per (query, provider) fan-out leg,
--                              carrying that leaf's hit count, latency, and
--                              degradation category. Lets Insights compute
--                              per-provider health with a GROUP BY instead of
--                              parsing a blob per row.
--
-- Times are RFC3339Nano strings, matching the registry/notes convention and the
-- wire format, so the window filter is a lexical string comparison.

CREATE TABLE IF NOT EXISTS query_telemetry (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  query_hash   TEXT NOT NULL,
  routed_types TEXT NOT NULL DEFAULT '',   -- comma-joined leaf types the query routed to
  result_count INTEGER NOT NULL DEFAULT 0, -- total hits across all providers
  zero_result  INTEGER NOT NULL DEFAULT 0, -- 1 when result_count = 0
  degraded     INTEGER NOT NULL DEFAULT 0, -- 1 when any provider degraded
  reranked     INTEGER NOT NULL DEFAULT 0, -- 1 when a unified rerank was produced
  auto_routed_external INTEGER NOT NULL DEFAULT 0, -- 1 when a web-shaped query auto-routed an external provider (OT-P2-002)
  escalated    INTEGER NOT NULL DEFAULT 0, -- 1 when an empty project corpus escalated to external (OT-P2-002)
  latency_ms   INTEGER NOT NULL DEFAULT 0,
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
  PRIMARY KEY (query_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_query_telemetry_provider_pid ON query_telemetry_provider(provider_id);
