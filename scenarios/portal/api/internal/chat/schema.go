package chat

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the chat domain's declarative SQLite schema.
func Schema() string { return schemaSQL }
