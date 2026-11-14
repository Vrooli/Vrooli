package recovery

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the sidecar checkpoint schema.
func Schema() string { return schemaSQL }
