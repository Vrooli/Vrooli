package database

import (
	"os"
	"strings"
)

// Dialect represents the database backend type.
type Dialect string

const (
	// DialectPostgres is the PostgreSQL dialect.
	DialectPostgres Dialect = "postgres"

	// DialectSQLite is the SQLite dialect.
	DialectSQLite Dialect = "sqlite"
)

// parseDialect parses the dialect from a string, defaulting to PostgreSQL.
func parseDialect(raw string) Dialect {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sqlite", "sqlite3":
		return DialectSQLite
	case "postgres", "postgresql", "":
		return DialectPostgres
	default:
		return DialectPostgres
	}
}

// getDialectFromEnv reads the database dialect from environment variables.
func getDialectFromEnv() Dialect {
	return parseDialect(os.Getenv("AM_DB_BACKEND"))
}
