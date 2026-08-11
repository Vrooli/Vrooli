package bindings

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the bindings domain schema for the central database.
func Schema() string { return schema }
