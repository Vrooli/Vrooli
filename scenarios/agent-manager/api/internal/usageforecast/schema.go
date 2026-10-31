package usageforecast

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema is the alert dedupe table owned by this package.
func Schema() string { return schemaSQL }
