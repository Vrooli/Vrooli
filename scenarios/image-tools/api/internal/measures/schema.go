package measures

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the measures domain's SQL contribution (the op_measure table),
// applied by database.EnsureSchemas via the modules.AllSchemas registry.
// Forward-only declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
func Schema() string { return schemaSQL }
