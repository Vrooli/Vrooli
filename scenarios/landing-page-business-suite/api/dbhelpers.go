package main

import (
	"database/sql"
)

// DialectHelper provides SQL dialect-specific expressions for databases
// that need to support both PostgreSQL and SQLite.
type DialectHelper struct {
	dialect string // "postgres" or "sqlite"
}

// NewDialectHelper creates a new dialect helper.
// If dialect is empty, defaults to "postgres".
func NewDialectHelper(dialect string) *DialectHelper {
	if dialect == "" {
		dialect = "postgres"
	}
	return &DialectHelper{dialect: dialect}
}

// NowExpr returns the appropriate SQL expression for current timestamp.
// PostgreSQL: NOW()
// SQLite: datetime('now')
func (d *DialectHelper) NowExpr() string {
	if d.dialect == "sqlite" {
		return "datetime('now')"
	}
	return "NOW()"
}

// Placeholder returns the appropriate placeholder for the given index.
// PostgreSQL uses $1, $2, etc.
// SQLite uses ?, but we use numbered placeholders for consistency.
// Note: For SQLite compatibility, consider using sqlx or named parameters.
func (d *DialectHelper) Placeholder(index int) string {
	// Currently both use PostgreSQL-style placeholders
	// as the database/sql driver handles translation
	return "$" + string(rune('0'+index))
}

// IsSQLite returns true if the dialect is SQLite.
func (d *DialectHelper) IsSQLite() bool {
	return d.dialect == "sqlite"
}

// IsPostgres returns true if the dialect is PostgreSQL.
func (d *DialectHelper) IsPostgres() bool {
	return d.dialect == "postgres" || d.dialect == ""
}

// Dialect returns the current dialect string.
func (d *DialectHelper) Dialect() string {
	return d.dialect
}

// NullStringValue extracts the value from a sql.NullString.
// Returns nil if the NullString is not valid, otherwise returns a pointer to the string value.
func NullStringValue(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// NullInt64Value extracts the value from a sql.NullInt64.
// Returns nil if the NullInt64 is not valid, otherwise returns a pointer to the int64 value.
func NullInt64Value(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}

// NullFloat64Value extracts the value from a sql.NullFloat64.
// Returns nil if the NullFloat64 is not valid, otherwise returns a pointer to the float64 value.
func NullFloat64Value(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	return &nf.Float64
}

// NullBoolValue extracts the value from a sql.NullBool.
// Returns nil if the NullBool is not valid, otherwise returns a pointer to the bool value.
func NullBoolValue(nb sql.NullBool) *bool {
	if !nb.Valid {
		return nil
	}
	return &nb.Bool
}

// StringToNullString converts a *string to sql.NullString.
// Returns an invalid NullString if the pointer is nil.
func StringToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// Int64ToNullInt64 converts a *int64 to sql.NullInt64.
// Returns an invalid NullInt64 if the pointer is nil.
func Int64ToNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}
