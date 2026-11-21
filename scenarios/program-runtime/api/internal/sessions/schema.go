package sessions

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the sessions domain schema for the central database bootstrap.
func Schema() string { return schema }
