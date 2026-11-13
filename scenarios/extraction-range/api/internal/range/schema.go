package rangedomain

import (
	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the range domain's idempotent storage contract.
func Schema() string { return schemaSQL }
