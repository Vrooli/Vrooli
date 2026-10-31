package retrieval

import _ "embed"

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }
