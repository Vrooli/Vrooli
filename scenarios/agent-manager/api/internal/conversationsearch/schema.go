package conversationsearch

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns this bounded context's idempotent, declarative projection
// schema. Every object is regenerable from Agent Manager's canonical events.
func Schema() string { return schemaSQL }
