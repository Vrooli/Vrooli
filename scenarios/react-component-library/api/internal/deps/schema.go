package deps

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the deps domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry.
// Idempotent (CREATE TABLE IF NOT EXISTS) so EnsureSchemas can re-run.
func Schema() string { return schemaSQL }
