package versions

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the versions domain's SQL contribution.
func Schema() string { return schemaSQL }
