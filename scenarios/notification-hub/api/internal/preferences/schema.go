package preferences

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the embedded preferences DDL, registered alongside the other
// domain schema providers in internal/modules.
func Schema() string { return schema }
