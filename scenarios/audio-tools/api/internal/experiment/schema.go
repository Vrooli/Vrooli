package experiment

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns this domain's declarative SQLite schema for
// database.EnsureSchemas via the modules.AllSchemas registry.
func Schema() string { return schemaSQL }
