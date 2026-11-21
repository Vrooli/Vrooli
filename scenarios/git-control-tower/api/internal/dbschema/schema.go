package dbschema

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the scenario's authoritative SQLite schema.
func Schema() string { return schemaSQL }
