package mints

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the mints domain's SQL contribution.
func Schema() string { return schemaSQL }
