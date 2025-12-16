package main

import (
	"context"
	"database/sql"
)

// DBChecker abstracts database health checks to enable testing without a real database.
// This is the seam for isolating database operations in health checks.
//
// Production code uses SQLDBChecker which wraps a *sql.DB.
// Test code can use FakeDBChecker (in db_checker_fake_test.go) to exercise
// health check logic without needing a running database.
//
// SEAM BOUNDARY: All database health operations should flow through this interface.
type DBChecker interface {
	// Ping checks if the database is reachable.
	Ping(ctx context.Context) error

	// IsConfigured returns true if the database is configured (non-nil).
	IsConfigured() bool
}

// SQLDBChecker implements DBChecker by wrapping a *sql.DB.
// This is the production implementation used when the API is running.
type SQLDBChecker struct {
	db *sql.DB
}

// NewSQLDBChecker creates a DBChecker from a *sql.DB.
// Returns a checker that reports unconfigured if db is nil.
func NewSQLDBChecker(db *sql.DB) DBChecker {
	return &SQLDBChecker{db: db}
}

// Ping checks if the database is reachable.
func (c *SQLDBChecker) Ping(ctx context.Context) error {
	if c.db == nil {
		return nil // IsConfigured() will return false anyway
	}
	return c.db.PingContext(ctx)
}

// IsConfigured returns true if the database handle is non-nil.
func (c *SQLDBChecker) IsConfigured() bool {
	return c.db != nil
}
