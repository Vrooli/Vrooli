package validation

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the validation domain's SQL contribution (terminal results and
// durable validation-operation checkpoints). Applied by database.EnsureSchemas.
// Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
