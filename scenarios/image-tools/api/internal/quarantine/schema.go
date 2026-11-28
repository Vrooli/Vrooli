package quarantine

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the quarantine SQL contribution, applied by
// database.EnsureSchemas through the modules.AllSchemas registry.
func Schema() string { return schemaSQL }
