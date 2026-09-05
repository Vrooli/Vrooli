package convergence

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the convergence domain's SQL contribution (the fitness-audit
// index: convergence_fitness + reference_health). Applied by
// database.EnsureSchemas via modules.AllSchemas(). Forward-only declarative —
// re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
func Schema() string { return schemaSQL }
