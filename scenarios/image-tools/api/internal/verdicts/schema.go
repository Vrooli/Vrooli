package verdicts

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the verdicts domain's SQL contribution (the job_rating table),
// applied by database.EnsureSchemas via the modules.AllSchemas registry.
func Schema() string { return schemaSQL }
