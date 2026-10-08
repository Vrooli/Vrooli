// Package docaccess owns the docaccess domain's storage schema.
package docaccess

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns this domain's DDL for database.EnsureSchemas.
func Schema() string { return schemaSQL }
