package proposals

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema is the proposal domain’s declarative storage contract.
func Schema() string { return schemaSQL }
