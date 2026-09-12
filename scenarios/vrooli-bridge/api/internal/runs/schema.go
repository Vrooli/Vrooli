package runs

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the runs domain's SQL contribution (the `runs` table and the
// append-only `run_events` table). Applied by database.EnsureSchemas via the
// modules.AllSchemas() registry. Forward-only declarative — re-runs are no-ops
// (CREATE TABLE IF NOT EXISTS). Never recreate the DB on a schema change.
func Schema() string { return schemaSQL }
