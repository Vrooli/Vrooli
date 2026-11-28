package byokstore

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema declares credentials owned by the BYOK persistence boundary.
func Schema() string { return schemaSQL }
