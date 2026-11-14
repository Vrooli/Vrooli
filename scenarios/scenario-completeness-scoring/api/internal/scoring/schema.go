package scoring

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the scoring domain's durable snapshot schema.
func Schema() string { return schemaSQL }
