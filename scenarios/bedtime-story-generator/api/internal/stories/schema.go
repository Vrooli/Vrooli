package stories

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the scenario-owned database schema.
func Schema() string { return schemaSQL }
