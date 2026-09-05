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

-- TM fixed-port ownership ledger — owned by internal/exposure/. One row per
-- scenario whose UI port Tunnel Manager switched from a range to a fixed port
-- (via structure-health) so it could be exposed as a scenario route. Revoke
-- releases the fixed port back to a range ONLY for scenarios recorded here, so
-- a hand-pinned fixed port is never reverted. Absence of a row means TM did not
-- assign the port (the safe default: never release it). CREATE TABLE IF NOT
-- EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS tm_port_assignments (
  scenario    TEXT PRIMARY KEY,
  assigned_at TEXT NOT NULL DEFAULT ''
);

-- Ingress ownership ledger — only records who owns each live ingress hostname.
-- Absence of a row means UNMANAGED and is never auto-removed.
CREATE TABLE IF NOT EXISTS ingress_ownership (
  hostname    TEXT PRIMARY KEY,
  owner       TEXT NOT NULL,
  scenario    TEXT NOT NULL DEFAULT '',
  note        TEXT NOT NULL DEFAULT '',
  adopted_at  TEXT NOT NULL DEFAULT ''
);

-- DNS ownership ledger — only records proxied CNAMEs Tunnel Manager created.
CREATE TABLE IF NOT EXISTS dns_ownership (
  hostname    TEXT PRIMARY KEY,
  record_id   TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL DEFAULT ''
);

-- Access ownership ledger — only records Access apps Tunnel Manager created.
CREATE TABLE IF NOT EXISTS access_ownership (
  host        TEXT PRIMARY KEY,
  app_id      TEXT NOT NULL DEFAULT '',
  policy_id   TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL DEFAULT ''
);
