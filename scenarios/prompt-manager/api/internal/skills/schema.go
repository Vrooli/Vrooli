package skills

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the skills domain's durable metrics table definitions.
func Schema() string { return schemaSQL }
