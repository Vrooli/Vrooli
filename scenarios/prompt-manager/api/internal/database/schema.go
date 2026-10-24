package database

import _ "embed"

//go:embed system.sql
var systemSchema string

func SystemSchema() string {
	return systemSchema
}
