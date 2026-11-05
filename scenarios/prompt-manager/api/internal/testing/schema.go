package testing

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the testing domain's durable table definitions.
func Schema() string { return schemaSQL }
