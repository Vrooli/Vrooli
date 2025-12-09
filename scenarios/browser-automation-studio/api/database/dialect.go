package database

import "strings"

// Dialect captures backend-specific decisions that affect value encoding/decoding.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

func parseDialect(raw string) Dialect {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sqlite":
		return DialectSQLite
	default:
		return DialectPostgres
	}
}

func (d Dialect) IsSQLite() bool   { return d == DialectSQLite }
func (d Dialect) IsPostgres() bool { return d == DialectPostgres }

// DialectProvider lets value types obtain the current dialect without relying on globals.
type DialectProvider interface {
	Dialect() Dialect
}
