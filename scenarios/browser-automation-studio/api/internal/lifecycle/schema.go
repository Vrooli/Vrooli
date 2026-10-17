package lifecycle

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the lifecycle trigger schema.
func Schema() string { return schemaSQL }
