package desktoplink

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the declarative schema owned by the desktop-link domain.
func Schema() string { return schema }
