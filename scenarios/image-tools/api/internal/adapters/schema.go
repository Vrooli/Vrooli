package adapters

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the adapters domain's SQL contribution (the enabled-state +
// install + custom overlay tables), applied by database.EnsureSchemas via the
// modules.AllSchemas registry. Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
