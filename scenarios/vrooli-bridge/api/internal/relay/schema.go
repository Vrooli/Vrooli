package relay

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the relay command journal schema. It is declarative and
// idempotent so command receipts remain available across Bridge restarts.
func Schema() string { return schemaSQL }
