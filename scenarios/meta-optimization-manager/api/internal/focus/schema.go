package focus

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the focus domain's SQL contribution (the gaps registry table).
// Applied by database.EnsureSchemas via modules.AllSchemas(). Forward-only
// declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
func Schema() string { return schemaSQL }
