package runannotations

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema is the run_annotations table owned by this package. It is applied
// idempotently by Store.EnsureSchema; the shared migration registry is not
// edited so this workstream does not serialize on a shared generated artifact.
func Schema() string { return schemaSQL }
