package trials

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the trials domain's SQL contribution (trials_runs +
// trial_gates). Applied by database.EnsureSchemas via modules.AllSchemas().
// Forward-only declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
func Schema() string { return schemaSQL }
