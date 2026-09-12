package redemption

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the redemption domain's SQL contribution.
func Schema() string { return schemaSQL }
