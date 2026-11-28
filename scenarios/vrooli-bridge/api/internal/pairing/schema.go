package pairing

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the pairing domain's SQL contribution, applied by
// database.EnsureSchemas via modules.AllSchemas(). Forward-only declarative —
// re-runs are no-ops (CREATE TABLE IF NOT EXISTS); never recreate on change.
func Schema() string { return schemaSQL }
