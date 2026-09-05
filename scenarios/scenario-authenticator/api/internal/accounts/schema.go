package accounts

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the accounts + realms SQL contribution, applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-only
// declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS / INSERT OR
// IGNORE). Additive columns land as a one-shot migration (storage-steer §5),
// never a DB recreate.
func Schema() string { return schemaSQL }
