package discovery

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the discovery domain's SQL contribution (the dismissals
// table), applied by database.EnsureSchemas via the modules.AllSchemas
// registry. Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
