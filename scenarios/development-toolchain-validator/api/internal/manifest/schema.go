package manifest

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the manifest domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry.
func Schema() string { return schemaSQL }
