package workflow

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the workflow domain schema applied during API startup.
func Schema() string { return schemaSQL }
