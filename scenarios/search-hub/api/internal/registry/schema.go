package registry

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the registry domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-
// only declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
//
// Adding a column lands as `ALTER TABLE providers ADD COLUMN IF NOT EXISTS
// <name> <type>` appended to schema.sql. Drops/renames need the brownfield
// migration path (deferred until the first scenario hits it).
func Schema() string { return schemaSQL }
