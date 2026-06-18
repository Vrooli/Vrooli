-- Cross-OS deployment gates + per-OS results — owned by internal/gate/.
-- Embedded by schema.go and applied via database.EnsureSchemas at boot (api/
-- main.go through modules.AllSchemas). Cross-OS deployment gate (OT-P1-002): one
-- durable Gate record per RunGate and its per-OS ledger. Times are RFC3339Nano
-- strings matching the wire format and the time.Time round-trip in sqlite.go.
-- args_json holds the validation verb's extra argument tokens as a JSON array.
-- Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops (migrate, never recreate).
-- Postgres-compatible column types for forward scale.
CREATE TABLE IF NOT EXISTS gates (
  id              TEXT PRIMARY KEY,
  scenario        TEXT NOT NULL DEFAULT '',
  target_revision TEXT NOT NULL DEFAULT '',
  verb            TEXT NOT NULL DEFAULT '',
  args_json       TEXT NOT NULL DEFAULT '[]',
  verdict         INTEGER NOT NULL DEFAULT 0,
  total_targets   INTEGER NOT NULL DEFAULT 0,
  passed          INTEGER NOT NULL DEFAULT 0,
  failed          INTEGER NOT NULL DEFAULT 0,
  pending         INTEGER NOT NULL DEFAULT 0,
  created_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gates_created_at ON gates(created_at DESC);

-- gate_os_results is the per-OS ledger, written once with the gate. The PENDING
-- rows carry a run_id the read path refreshes against the live run state; the
-- (gate_id, os) pair is unique (one line per target OS).
CREATE TABLE IF NOT EXISTS gate_os_results (
  gate_id     TEXT NOT NULL,
  os          TEXT NOT NULL,
  node_id     TEXT NOT NULL DEFAULT '',
  run_id      TEXT NOT NULL DEFAULT '',
  disposition INTEGER NOT NULL DEFAULT 0,
  exit_code   INTEGER NOT NULL DEFAULT 0,
  detail      TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (gate_id, os)
);

CREATE INDEX IF NOT EXISTS idx_gate_os_results_gate ON gate_os_results(gate_id, os);
