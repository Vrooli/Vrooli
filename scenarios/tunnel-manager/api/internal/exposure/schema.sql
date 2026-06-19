-- Leases table — owned by internal/exposure/. One row per on-demand
-- exposure grant: which scenario, who asked, when it expires, how many
-- times it was extended, and its lifecycle status. CORE exposure is NOT
-- recorded here — it is derived from the api-core/coreset closure at
-- reconcile time. Embedded by schema.go and applied via
-- database.EnsureSchemas at boot. Times are RFC3339Nano strings matching
-- the wire format and the time.Time round-trip in sqlite.go::scanLease.
-- CREATE TABLE IF NOT EXISTS so re-runs are no-ops; add columns with
-- ALTER TABLE ... ADD COLUMN (migrate, never recreate).
CREATE TABLE IF NOT EXISTS leases (
  id             TEXT PRIMARY KEY,
  scenario       TEXT NOT NULL,
  requested_by   TEXT NOT NULL DEFAULT '',
  created_at     TEXT NOT NULL,
  expires_at     TEXT NOT NULL,
  extended_count INTEGER NOT NULL DEFAULT 0,
  status         TEXT NOT NULL DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_leases_scenario ON leases(scenario);
CREATE INDEX IF NOT EXISTS idx_leases_status ON leases(status);
