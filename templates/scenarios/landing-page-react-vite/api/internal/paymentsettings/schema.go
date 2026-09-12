package paymentsettings

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the payment settings domain's SQL contribution, applied by
// database.EnsureSchemas via the modules registry.
func Schema() string { return schemaSQL }
