package workspaces

import (
	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the workspaces domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry and re-applied
// by Migrate before importing legacy rows, so both entry points must be safe
// to call in either order.
func Schema() string { return schemaSQL }
