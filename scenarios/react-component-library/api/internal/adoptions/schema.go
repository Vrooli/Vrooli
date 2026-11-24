package adoptions

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the adoptions domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. It describes
// the current shape for fresh databases; explicit legacy repair lives in
// EnsureMigrations and runs during API startup after schema application.
func Schema() string { return schemaSQL }
