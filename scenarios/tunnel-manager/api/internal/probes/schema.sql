-- Probes table — owned by internal/probes/. Append-only liveness probe
-- history: every probe cycle records one internal + one external row per
-- enabled route. Embedded by schema.go and applied via
-- database.EnsureSchemas at boot (api/main.go through the
-- modules.AllSchemas registry). Times are stored as RFC3339Nano strings
-- matching the wire format and the time.Time round-trip in
-- sqlite.go::scanProbe. Use CREATE TABLE IF NOT EXISTS so re-runs are
-- no-ops; add columns with ALTER TABLE ... ADD COLUMN (migrate, never
-- recreate).
CREATE TABLE IF NOT EXISTS probes (
  id          TEXT PRIMARY KEY,
  subdomain   TEXT NOT NULL,
  kind        TEXT NOT NULL,
  status      TEXT NOT NULL,
  latency_ms  INTEGER NOT NULL DEFAULT 0,
  status_code INTEGER NOT NULL DEFAULT 0,
  error_msg   TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_probes_subdomain ON probes(subdomain);
CREATE INDEX IF NOT EXISTS idx_probes_subdomain_kind_created ON probes(subdomain, kind, created_at);
