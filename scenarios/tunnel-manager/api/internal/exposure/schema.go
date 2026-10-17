package exposure

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the exposure domain's SQL contribution (the leases
// table). Applied by database.EnsureSchemas via modules.AllSchemas().
// Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
