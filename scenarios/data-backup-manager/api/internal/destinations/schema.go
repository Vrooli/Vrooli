package destinations

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the destinations domain's SQL contribution, applied by
// database.EnsureSchemas via the modules registry. Forward-only declarative —
// re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
func Schema() string { return schemaSQL }
