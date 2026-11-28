package jobs

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the jobs domain-owned SQLite schema. It is declarative and
// safe to apply more than once; module registration remains a separate owner.
func Schema() string { return schemaSQL }
