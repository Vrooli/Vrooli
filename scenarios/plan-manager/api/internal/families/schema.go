package families

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema is the families domain's declarative database contribution.
func Schema() string { return schemaSQL }
