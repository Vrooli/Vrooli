package authoring

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the authoring domain's SQL contribution (the authoring_sessions
// table). Applied by database.EnsureSchemas via the modules.AllSchemas()
// registry. Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
