package privacy

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the idempotent privacy domain schema.
func Schema() string { return schemaSQL }
