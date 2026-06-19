-- Metrics table — owned by internal/tunnel/. The scraped cloudflared
-- Prometheus time-series: one row per scrape, so the UI can render trends and
-- recovery can detect degraded mode (HA connections dropping, RTT spikes).
-- Embedded by schema.go and applied via database.EnsureSchemas at boot
-- (api/main.go through the modules.AllSchemas registry). Times are stored as
-- RFC3339Nano strings matching the wire format and the time.Time round-trip in
-- sqlite.go::scanSample. Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops;
-- add columns with ALTER TABLE ... ADD COLUMN (migrate, never recreate).
CREATE TABLE IF NOT EXISTS metrics (
  id              TEXT PRIMARY KEY,
  ha_connections  INTEGER NOT NULL DEFAULT 0,
  request_errors  REAL NOT NULL DEFAULT 0,
  active_streams  INTEGER NOT NULL DEFAULT 0,
  smoothed_rtt_ms REAL NOT NULL DEFAULT 0,
  scraped_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_metrics_scraped_at ON metrics(scraped_at);
