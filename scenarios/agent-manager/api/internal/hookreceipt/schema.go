package hookreceipt

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema is the durable hook receipt schema owned by this package.
func Schema() string { return schemaSQL }
