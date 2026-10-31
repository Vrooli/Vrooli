package settings

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema declares provider-routing and voice override settings.
func Schema() string { return schemaSQL }
