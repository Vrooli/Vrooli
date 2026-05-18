package golden

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the golden domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry.
func Schema() string { return schemaSQL }
