package coverage

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the coverage domain's SQL contribution (the short-TTL snapshot
// cache table). Applied by database.EnsureSchemas via modules.AllSchemas().
// Forward-only declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
func Schema() string { return schemaSQL }
