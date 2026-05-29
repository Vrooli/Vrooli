package migration

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the migration domain's CREATE TABLE statements for
// apidb.EnsureSchemas. Registered in modules.AllSchemas alongside the
// migration handler (Phase 5).
func Schema() string { return schemaSQL }
