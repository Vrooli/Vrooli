package plans

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the plans domain's SQL contribution (the plans + plan_edges
// tables). Applied by database.EnsureSchemas via the modules.AllSchemas()
// registry. Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
