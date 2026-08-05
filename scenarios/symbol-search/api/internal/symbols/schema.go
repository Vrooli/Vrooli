package symbols

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the symbols domain schema applied during API startup.
func Schema() string { return schemaSQL }
