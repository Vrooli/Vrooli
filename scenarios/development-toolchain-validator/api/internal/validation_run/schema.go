package validation_run

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the validation_run domain's SQL contribution.
func Schema() string { return schemaSQL }
