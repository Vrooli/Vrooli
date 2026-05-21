package graph

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the graph domain's SQL contribution.
func Schema() string { return schemaSQL }
