package core

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the core domain schema.
func Schema() string { return schemaSQL }
