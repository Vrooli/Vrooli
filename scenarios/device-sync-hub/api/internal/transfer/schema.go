package transfer

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the transfer domain's SQL contribution. Applied by
// database.EnsureSchemas via modules.AllSchemas(). Forward-only declarative —
// re-runs are no-ops (CREATE TABLE IF NOT EXISTS). Column additions append an
// `ALTER TABLE ... ADD COLUMN`; drops/renames take the brownfield migration
// path (never recreate the DB on a schema change).
func Schema() string { return schemaSQL }
