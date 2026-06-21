-- Config table — owned by internal/config/. The tunnel's management
-- configuration: a single logical row (singleton keyed on id='singleton')
-- holding the management mode plus Cloudflare tunnel coordinates. Embedded
-- by schema.go and applied via database.EnsureSchemas at boot (api/main.go
-- through the modules.AllSchemas registry). Use CREATE TABLE IF NOT EXISTS
-- so re-runs are no-ops; add columns with ALTER TABLE ... ADD COLUMN
-- (migrate, never recreate).
CREATE TABLE IF NOT EXISTS tunnel_config (
  id            TEXT PRIMARY KEY DEFAULT 'singleton',
  mode          TEXT NOT NULL DEFAULT 'local',
  tunnel_id     TEXT NOT NULL DEFAULT '',
  account_id    TEXT NOT NULL DEFAULT '',
  cred_ref      TEXT NOT NULL DEFAULT '',
  prom_endpoint TEXT NOT NULL DEFAULT '127.0.0.1:20241'
);

-- Ingress ownership ledger — owned by internal/config/. The authoritative
-- record of who owns each live ingress hostname, so reconciliation can tell
-- managed entries (TM created them) from external-acknowledged and ignored
-- ones — without guessing. Keyed on the full hostname (<subdomain>.<domain>),
-- not the subdomain, because external routes may hang off other apexes.
-- Absence of a row means UNMANAGED (the safe default: surface as drift,
-- never auto-remove). owner ∈ {MANAGED, EXTERNAL, IGNORED}. Times are
-- RFC3339Nano strings matching the round-trip in ledger_sqlite.go. Use
-- CREATE TABLE IF NOT EXISTS so re-runs are no-ops; add columns with
-- ALTER TABLE ... ADD COLUMN (migrate, never recreate).
CREATE TABLE IF NOT EXISTS ingress_ownership (
  hostname    TEXT PRIMARY KEY,
  owner       TEXT NOT NULL,
  scenario    TEXT NOT NULL DEFAULT '',
  note        TEXT NOT NULL DEFAULT '',
  adopted_at  TEXT NOT NULL DEFAULT ''
);
