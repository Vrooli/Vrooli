package admin

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the admin domain's SQL contribution (admin_users), applied by
// database.EnsureSchemas via the modules registry.
func Schema() string { return schemaSQL }
