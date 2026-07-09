package catalog

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the template-manager catalog schema. It is intentionally
// forward-only and idempotent so startup can run it on every boot.
func Schema() string { return schemaSQL }
