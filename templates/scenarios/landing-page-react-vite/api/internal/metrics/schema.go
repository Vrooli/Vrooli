package metrics

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the metrics domain's SQL contribution, applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. It must run
// after the variant schema, which it foreign-keys.
func Schema() string { return schemaSQL }
