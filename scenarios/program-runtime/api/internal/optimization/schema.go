package optimization

import _ "embed"

//go:embed schema.sql
var schema string

// Schema exposes the optimizer's durable run journal to the scenario schema
// owner, which applies it through the configured central database.
func Schema() string { return schema }
