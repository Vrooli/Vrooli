-- Audit records table — owned by internal/audit/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot. The append-only accountability
-- trail (OT-P0-008). This table is append-only by construction: sqlite.go only
-- ever INSERTs and SELECTs — there is no UPDATE or DELETE statement anywhere in
-- the domain, so a record cannot be mutated after it is written. Times are
-- RFC3339Nano strings; args is a JSON-encoded string array. Use CREATE TABLE IF
-- NOT EXISTS so re-runs are no-ops (migrate, never recreate). Postgres-
-- compatible column types for forward scale.
CREATE TABLE IF NOT EXISTS audit_records (
  id          TEXT PRIMARY KEY,
  action      INTEGER NOT NULL DEFAULT 0,
  actor       TEXT NOT NULL DEFAULT '',
  node_id     TEXT NOT NULL DEFAULT '',
  scenario    TEXT NOT NULL DEFAULT '',
  verb        TEXT NOT NULL DEFAULT '',
  args        TEXT NOT NULL DEFAULT '[]',
  outcome     INTEGER NOT NULL DEFAULT 0,
  detail      TEXT NOT NULL DEFAULT '',
  run_id      TEXT NOT NULL DEFAULT '',
  recorded_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_recorded_at ON audit_records(recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_node ON audit_records(node_id);
CREATE INDEX IF NOT EXISTS idx_audit_run ON audit_records(run_id);
