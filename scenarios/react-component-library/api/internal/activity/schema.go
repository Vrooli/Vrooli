package activity

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the activity domain's SQL contribution. The modules registry
// adds it to EnsureSchemas, so the table has exactly one owner: this package.
func Schema() string { return schemaSQL }
