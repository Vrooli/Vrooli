package businessaccount

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the declarative schema owned by the business-account domain.
func Schema() string { return schema }
