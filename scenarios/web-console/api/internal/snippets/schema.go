package snippets

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the idempotent SQL owned by the snippets domain.
func Schema() string { return schemaSQL }
