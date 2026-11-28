package recovery

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the recovery domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry.
// Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
