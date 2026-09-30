package assets

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the assets domain's SQL contribution (assets), applied by
// database.EnsureSchemas via the modules registry.
func Schema() string { return schemaSQL }
