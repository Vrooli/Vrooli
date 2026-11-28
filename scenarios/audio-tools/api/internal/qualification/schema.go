package trustfloor

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the shared product schema for persisted qualification
// evidence. Lab runs produce evidence, while product model promotion reads it.
func Schema() string { return schemaSQL }
