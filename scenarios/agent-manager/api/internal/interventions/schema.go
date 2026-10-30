package interventions

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema is the operator_interventions table owned by this package.
func Schema() string { return schemaSQL }
