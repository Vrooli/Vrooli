package variant

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the variant domain's SQL contribution, applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. It must run
// before the content domain, which foreign-keys the variants table.
func Schema() string { return schemaSQL }
