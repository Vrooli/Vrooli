package branding

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the branding domain's SQL contribution, applied by
// database.EnsureSchemas via the modules.AllSchemas() registry.
func Schema() string { return schemaSQL }
