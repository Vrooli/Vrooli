// Package validationcache persists sanitized, normalized scanner evidence for
// the validation domain. It contains no scanner execution or policy logic.
package validationcache

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the validation evidence cache's declarative schema.
func Schema() string { return schemaSQL }
