package capabilities

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the capability domain DDL for EnsureSchemas.
func Schema() string { return schemaSQL }
