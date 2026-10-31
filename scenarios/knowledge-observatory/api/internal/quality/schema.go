// Package quality owns the quality domain's storage schema.
package quality

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns this domain's DDL for database.EnsureSchemas.
func Schema() string { return schemaSQL }
