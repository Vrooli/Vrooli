package artifact

import _ "embed"

//go:embed schema.sql
var schema string

// Schema is the durable metadata substrate for generated run artifacts.
func Schema() string { return schema }
