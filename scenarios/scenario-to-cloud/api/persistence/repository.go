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
	"strings"
)

// DB is the small SQL surface used by this repository. Both *sql.DB and
// database.RoutedDB implement it, which keeps every query on the request
// context and allows test-genie to route shadow runs without giving the
// repository a second persistence path.
type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Repository provides database operations for the deployment domain.
type Repository struct {
	db DB
}

// NewRepository creates a new repository with the given database connection.
func NewRepository(db DB) *Repository {
	return &Repository{db: db}
}

// DB returns the repository's routed SQL surface for direct access when
// needed. Callers must pass their request context so test-genie routing is
// preserved.
func (r *Repository) DB() DB {
	return r.db
}

// InitSchema initializes the database schema for the scenario-to-cloud scenario.
// This creates all required tables and indexes using idempotent statements.
// Note: With per-scenario databases, we use the public schema directly.
func (r *Repository) InitSchema(ctx context.Context) error {
	return r.InitSchemaOn(ctx, r.db)
}

// InitSchemaOn applies the idempotent schema and migrations to a specific
// pool. It is used both for the primary pool and as RoutedDB's test-pool
// initializer, so a shadow database can never begin a run with a partial
// schema.
func (r *Repository) InitSchemaOn(ctx context.Context, db DB) error {
	return r.InitSchemaOnDialect(ctx, db, "postgres")
}

// InitSchemaOnDialect applies the schema for the selected SQL engine. The
// production authority is PostgreSQL, while Test Genie deliberately provisions
// disposable SQLite pools for routed workflow runs. Keeping the two DDL forms
// explicit prevents PostgreSQL-only defaults and casts from making the test
// isolation contract fail before a workflow can start.
func (r *Repository) InitSchemaOnDialect(ctx context.Context, db DB, dialect string) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}
	if strings.EqualFold(strings.TrimSpace(dialect), "sqlite") || strings.EqualFold(strings.TrimSpace(dialect), "sqlite3") {
		return initSQLiteSchema(ctx, db)
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
		preflight_result JSONB,
		last_inspect_result JSONB,
		ssh_identity JSONB,

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

	CREATE TABLE IF NOT EXISTS cloud_instances (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL UNIQUE,
		provider TEXT NOT NULL,
		state TEXT NOT NULL,
		image TEXT NOT NULL,
		workdir TEXT NOT NULL,
		address TEXT NOT NULL DEFAULT '127.0.0.1',
		ssh_port INTEGER NOT NULL DEFAULT 0,
		profile TEXT NOT NULL DEFAULT 'headless-linux',
		pid INTEGER NOT NULL DEFAULT 0,
		command JSONB NOT NULL DEFAULT '[]'::jsonb,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		last_error TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_cloud_instances_state ON cloud_instances(state);
	`

	if _, err := db.ExecContext(ctx, baseSchema); err != nil {
		return fmt.Errorf("failed to create base schema: %w", err)
	}

	// Run any migrations (future-proofing)
	migrations := []struct {
		name string
		sql  string
	}{
		// Add migrations here as schema evolves
		{"add_progress_fields", `
			ALTER TABLE deployments ADD COLUMN IF NOT EXISTS progress_step TEXT;
			ALTER TABLE deployments ADD COLUMN IF NOT EXISTS progress_percent REAL DEFAULT 0;
		`},
		{"add_deployment_history", `
			ALTER TABLE deployments ADD COLUMN IF NOT EXISTS deployment_history JSONB DEFAULT '[]'::jsonb;
		`},
		{"add_investigations_table", `
			CREATE TABLE IF NOT EXISTS deployment_investigations (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
				deployment_run_id TEXT,
				status TEXT NOT NULL DEFAULT 'pending'
					CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
				findings TEXT,
				progress INTEGER DEFAULT 0,
				details JSONB,
				agent_run_id TEXT,
				error_message TEXT,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				completed_at TIMESTAMPTZ
			);
			CREATE INDEX IF NOT EXISTS idx_investigations_deployment_id ON deployment_investigations(deployment_id);
			CREATE INDEX IF NOT EXISTS idx_investigations_status ON deployment_investigations(status);
		`},
		{"add_investigations_run_id", `
			ALTER TABLE deployment_investigations
			ADD COLUMN IF NOT EXISTS deployment_run_id TEXT;
		`},
		{"add_idempotency_tracking", `
			-- Track completed steps for replay-safe execution
			ALTER TABLE deployments ADD COLUMN IF NOT EXISTS completed_steps JSONB DEFAULT '[]'::jsonb;
			-- UUID for the current execution run; changes on each fresh execution
			ALTER TABLE deployments ADD COLUMN IF NOT EXISTS run_id TEXT;
		`},
		{"add_preflight_result", `
			ALTER TABLE deployments ADD COLUMN IF NOT EXISTS preflight_result JSONB;
		`},
		{"add_ssh_identity", `
			ALTER TABLE deployments ADD COLUMN IF NOT EXISTS ssh_identity JSONB;
		`},
		{"add_cloud_instance_runtime_fields", `
			ALTER TABLE cloud_instances ADD COLUMN IF NOT EXISTS address TEXT NOT NULL DEFAULT '127.0.0.1';
			ALTER TABLE cloud_instances ADD COLUMN IF NOT EXISTS ssh_port INTEGER NOT NULL DEFAULT 0;
			ALTER TABLE cloud_instances ADD COLUMN IF NOT EXISTS profile TEXT NOT NULL DEFAULT 'headless-linux';
		`},
		{"add_task_columns", `
			-- Add columns for unified task system (investigate/fix tasks)
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS task_type TEXT DEFAULT 'investigate';
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS focus_harness BOOLEAN DEFAULT true;
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS focus_subject BOOLEAN DEFAULT true;
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS effort TEXT;
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS perm_immediate BOOLEAN DEFAULT false;
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS perm_permanent BOOLEAN DEFAULT false;
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS perm_prevention BOOLEAN DEFAULT false;
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS iteration INTEGER DEFAULT 0;
			ALTER TABLE deployment_investigations
				ADD COLUMN IF NOT EXISTS max_iterations INTEGER DEFAULT 5;
		`},
	}

	for _, m := range migrations {
		if _, err := db.ExecContext(ctx, m.sql); err != nil {
			// Ignore "already exists" type errors
			errStr := err.Error()
			if !contains(errStr, "already exists") && !contains(errStr, "duplicate") {
				return fmt.Errorf("migration %q failed: %w", m.name, err)
			}
		}
	}

	return nil
}

// initSQLiteSchema is intentionally complete rather than migration-shaped:
// each routed Test Genie pool is fresh and SQLite does not support the
// PostgreSQL casts/functions used by the production DDL. All columns required
// by the repository are declared here so the shadow pool has the same logical
// surface as the primary database.
func initSQLiteSchema(ctx context.Context, db DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS deployments (
	 id TEXT PRIMARY KEY,
	 name TEXT NOT NULL,
	 scenario_id TEXT NOT NULL,
	 status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'setup_running', 'setup_complete', 'deploying', 'deployed', 'failed', 'stopped')),
	 manifest TEXT NOT NULL,
	 bundle_path TEXT,
	 bundle_sha256 TEXT,
	 bundle_size_bytes INTEGER,
	 setup_result TEXT,
	 deploy_result TEXT,
	 preflight_result TEXT,
	 last_inspect_result TEXT,
	 ssh_identity TEXT,
	 error_message TEXT,
	 error_step TEXT,
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 last_deployed_at TIMESTAMP,
	 last_inspected_at TIMESTAMP,
	 progress_step TEXT,
	 progress_percent REAL DEFAULT 0,
	 deployment_history TEXT DEFAULT '[]',
	 completed_steps TEXT DEFAULT '[]',
	 run_id TEXT
);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
CREATE INDEX IF NOT EXISTS idx_deployments_scenario_id ON deployments(scenario_id);
CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON deployments(created_at DESC);

CREATE TABLE IF NOT EXISTS cloud_instances (
	 id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	 name TEXT NOT NULL UNIQUE,
	 provider TEXT NOT NULL,
	 state TEXT NOT NULL,
	 image TEXT NOT NULL,
	 workdir TEXT NOT NULL,
	 address TEXT NOT NULL DEFAULT '127.0.0.1',
	 ssh_port INTEGER NOT NULL DEFAULT 0,
	 profile TEXT NOT NULL DEFAULT 'headless-linux',
	 pid INTEGER NOT NULL DEFAULT 0,
	 command TEXT NOT NULL DEFAULT '[]',
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 last_error TEXT
);
CREATE INDEX IF NOT EXISTS idx_cloud_instances_state ON cloud_instances(state);

CREATE TABLE IF NOT EXISTS deployment_investigations (
	 id TEXT PRIMARY KEY,
	 deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	 deployment_run_id TEXT,
	 status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
	 findings TEXT,
	 progress INTEGER DEFAULT 0,
	 details TEXT,
	 agent_run_id TEXT,
	 error_message TEXT,
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 completed_at TIMESTAMP,
	 task_type TEXT DEFAULT 'investigate',
	 focus_harness INTEGER DEFAULT 1,
	 focus_subject INTEGER DEFAULT 1,
	 effort TEXT,
	 perm_immediate INTEGER DEFAULT 0,
	 perm_permanent INTEGER DEFAULT 0,
	 perm_prevention INTEGER DEFAULT 0,
	 iteration INTEGER DEFAULT 0,
	 max_iterations INTEGER DEFAULT 5
);
CREATE INDEX IF NOT EXISTS idx_investigations_deployment_id ON deployment_investigations(deployment_id);
CREATE INDEX IF NOT EXISTS idx_investigations_status ON deployment_investigations(status);
`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("failed to create SQLite schema: %w", err)
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
