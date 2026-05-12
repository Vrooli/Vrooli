package components

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the components domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry.
// Forward-only declarative — re-runs are no-ops (CREATE TABLE IF NOT
// EXISTS). New columns land as ALTER TABLE … ADD COLUMN IF NOT EXISTS
// appended to schema.sql.
func Schema() string { return schemaSQL }
