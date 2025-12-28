// Package persistence provides database operations for the scenario-to-cloud scenario.
// This package centralizes all database access for deployment records.
//
// File organization:
//   - repository.go: Base repository and schema initialization
//   - deployment.go: Deployment CRUD operations
package persistence

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository provides database operations for the deployment domain.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository with the given database connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying database connection for direct access when needed.
func (r *Repository) DB() *sql.DB {
	return r.db
}

// InitSchema initializes the database schema for the scenario-to-cloud scenario.
// This creates all required tables and indexes using idempotent statements.
func (r *Repository) InitSchema(ctx context.Context) error {
	schemaName := "scenario_to_cloud"

	// Create scenario-specific schema
	if _, err := r.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName)); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Set search_path at database level so all connections use it
	var dbname string
	if err := r.db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbname); err == nil {
		if _, err := r.db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s SET search_path TO %s, public", dbname, schemaName)); err != nil {
			// Ignore error - may not have ALTER DATABASE permission
		}
	}

	// Set for current session
	if _, err := r.db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName)); err != nil {
		return fmt.Errorf("failed to set session search_path: %w", err)
	}

	// Create deployments table
	baseSchema := `
	CREATE TABLE IF NOT EXISTS deployments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

		-- Identification
		name TEXT NOT NULL,
		scenario_id TEXT NOT NULL,

		-- Status
		status TEXT NOT NULL DEFAULT 'pending'
			CHECK (status IN ('pending', 'setup_running', 'setup_complete',
							  'deploying', 'deployed', 'failed', 'stopped')),

		-- Manifest snapshot (full JSON for reproducibility)
		manifest JSONB NOT NULL,

		-- Bundle info
		bundle_path TEXT,
		bundle_sha256 TEXT,
		bundle_size_bytes BIGINT,

		-- Results (JSONB for flexibility)
		setup_result JSONB,
		deploy_result JSONB,
		last_inspect_result JSONB,

		-- Error tracking
		error_message TEXT,
		error_step TEXT,

		-- Timestamps
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		last_deployed_at TIMESTAMPTZ,
		last_inspected_at TIMESTAMPTZ
	);

	CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
	CREATE INDEX IF NOT EXISTS idx_deployments_scenario_id ON deployments(scenario_id);
	CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON deployments(created_at DESC);
	`

	if _, err := r.db.ExecContext(ctx, baseSchema); err != nil {
		return fmt.Errorf("failed to create base schema: %w", err)
	}

	// Run any migrations (future-proofing)
	migrations := []struct {
		name string
		sql  string
	}{
		// Add migrations here as schema evolves
		// {"add some_column", `ALTER TABLE deployments ADD COLUMN IF NOT EXISTS some_column TEXT`},
	}

	for _, m := range migrations {
		if _, err := r.db.ExecContext(ctx, m.sql); err != nil {
			// Ignore "already exists" type errors
			errStr := err.Error()
			if !contains(errStr, "already exists") && !contains(errStr, "duplicate") {
				return fmt.Errorf("migration %q failed: %w", m.name, err)
			}
		}
	}

	return nil
}

// contains checks if a string contains a substring (case-insensitive check not needed here)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
