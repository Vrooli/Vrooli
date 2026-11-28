package vault

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the schema owned by the password-manager vault domain.
func Schema() string { return schemaSQL }
