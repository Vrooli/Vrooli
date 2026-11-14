package authorization

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the authorization storage schema.
func Schema() string { return schemaSQL }
