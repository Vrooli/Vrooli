package dependencies

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the dependencies domain's SQL contribution, applied via
// database.EnsureSchemas through the modules.AllSchemas() registry. Forward-
// only declarative (CREATE TABLE IF NOT EXISTS).
func Schema() string { return schemaSQL }
