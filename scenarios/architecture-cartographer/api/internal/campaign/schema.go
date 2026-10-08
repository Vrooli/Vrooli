package campaign

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the campaign domain's CREATE TABLE statements for
// apidb.EnsureSchemas. Registered in modules.AllSchemas alongside the
// campaign handler.
func Schema() string { return schemaSQL }
