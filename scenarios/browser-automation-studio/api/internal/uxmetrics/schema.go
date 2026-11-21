package uxmetrics

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the UX metrics domain schema.
func Schema() string { return schemaSQL }
