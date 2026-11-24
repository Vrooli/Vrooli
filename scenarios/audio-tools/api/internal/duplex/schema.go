package duplex

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema declares the durable correlation journal for LPBS-owned duplex holds.
func Schema() string { return schemaSQL }
