package execution

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the execution domain's SQL contribution (the executions /
// decisions / findings / handoffs / velocity_points tables). Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-only
// declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }
