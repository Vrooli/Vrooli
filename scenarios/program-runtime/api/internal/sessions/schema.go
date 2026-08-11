package sessions

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the sessions domain schema for the central database
// bootstrap. The SQL is idempotent so lifecycle restarts are safe.
func Schema() string { return schema }
