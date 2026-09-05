package gate

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the gate domain's SQL contribution (the `gates` table and the
// per-OS `gate_os_results` ledger). Applied by database.EnsureSchemas via the
// modules.AllSchemas() registry. Forward-only declarative — re-runs are no-ops
// (CREATE TABLE IF NOT EXISTS). Never recreate the DB on a schema change.
func Schema() string { return schemaSQL }
