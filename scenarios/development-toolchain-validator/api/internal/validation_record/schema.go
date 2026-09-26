package validation_record

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the validation_record domain's SQL contribution.
func Schema() string { return schemaSQL }
