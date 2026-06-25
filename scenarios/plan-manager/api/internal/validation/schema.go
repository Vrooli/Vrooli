package validation

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the validation domain's SQL contribution (the validation_results
// table — the last-known result per plan/phase that the execution context server
// reads). Applied by database.EnsureSchemas via the modules.AllSchemas() registry.
// Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
