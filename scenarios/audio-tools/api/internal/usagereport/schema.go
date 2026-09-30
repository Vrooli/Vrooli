package usagereport

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema declares durable provider-usage accounting rows.
func Schema() string { return schemaSQL }
