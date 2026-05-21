package manifest

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the manifest domain's SQL contribution.
func Schema() string { return schemaSQL }
