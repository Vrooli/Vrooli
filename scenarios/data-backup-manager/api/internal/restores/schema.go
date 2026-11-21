package restores

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the restores domain's SQL contribution, applied by
// database.EnsureSchemas via the modulecatalog.AllSchemas registry. Forward-only
// declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
