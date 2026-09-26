package audits

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the audits domain's SQL contribution, applied by
// database.EnsureSchemas via the modules.AllSchemas registry. Forward-only
// declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
