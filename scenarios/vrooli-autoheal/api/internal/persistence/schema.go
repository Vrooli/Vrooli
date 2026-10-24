package persistence

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the authoritative SQLite persistence schema.
func Schema() string { return schemaSQL }
