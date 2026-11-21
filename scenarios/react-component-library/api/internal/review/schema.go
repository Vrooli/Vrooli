package review

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the review domain's SQL contribution. findings and
// finding_dispositions have one owner: this package.
func Schema() string { return schemaSQL }
