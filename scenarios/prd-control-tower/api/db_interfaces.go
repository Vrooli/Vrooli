package main

import "database/sql"

// rowScanner represents the ability to scan database row results
type rowScanner interface {
	Scan(dest ...any) error
}

// dbQueryExecutor represents a minimal database interface for queries and execution
type dbQueryExecutor interface {
	QueryRow(query string, args ...any) rowScanner
	Exec(query string, args ...any) (sql.Result, error)
}

// sqlDB wraps *sql.DB to implement dbQueryExecutor
type sqlDB struct {
	db *sql.DB
}

func (s sqlDB) QueryRow(query string, args ...any) rowScanner {
	return s.db.QueryRow(query, args...)
}

func (s sqlDB) Exec(query string, args ...any) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

// dbExecutor represents the ability to execute database commands
type dbExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
}
