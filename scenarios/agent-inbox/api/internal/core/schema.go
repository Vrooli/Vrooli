package core

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the core domain schema applied during API startup.
func Schema() string { return schemaSQL }
