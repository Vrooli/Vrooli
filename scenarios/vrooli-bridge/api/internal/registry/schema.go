package registry

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the registry domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-only
// declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS). Column
// additions append an `ALTER TABLE nodes ADD COLUMN`; drops/renames take the
// brownfield migration path (never recreate the DB on a schema change).
func Schema() string { return schemaSQL }
