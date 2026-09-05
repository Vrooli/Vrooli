package studio

import _ "embed"

//go:embed schema.sql
var schema string

// Schema is registered at API boot through the scenario's domain schema seam.
func Schema() string { return schema }
