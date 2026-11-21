package database

import _ "embed"

//go:embed system.sql
var systemSchemaSQL string

// SystemSchema returns cross-cutting SQL that does not belong to a domain.
func SystemSchema() string { return systemSchemaSQL }
