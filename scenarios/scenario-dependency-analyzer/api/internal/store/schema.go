package store

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the store domain's SQLite schema.
func Schema() string { return schemaSQL }
