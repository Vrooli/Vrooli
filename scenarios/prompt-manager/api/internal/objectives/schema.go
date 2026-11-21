package objectives

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the objectives domain's durable table definitions.
func Schema() string { return schemaSQL }
