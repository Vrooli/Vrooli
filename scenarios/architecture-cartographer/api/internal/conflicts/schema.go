package conflicts

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the conflicts domain's SQL contribution.
func Schema() string { return schemaSQL }
