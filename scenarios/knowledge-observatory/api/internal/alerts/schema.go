// Package alerts owns the alerts domain's storage schema.
package alerts

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns this domain's DDL for database.EnsureSchemas.
func Schema() string { return schemaSQL }
