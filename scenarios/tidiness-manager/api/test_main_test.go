package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testContainer *postgres.PostgresContainer
	testDB        *sql.DB
)

// TestMain spins up a disposable Postgres for integration tests when DATABASE_URL
// is not already provided by the environment.
func TestMain(m *testing.M) {
	if os.Getenv("API_PORT") == "" {
		os.Setenv("API_PORT", "18080")
	}

	// If a DATABASE_URL is already provided (e.g., CI), reuse it.
	if os.Getenv("DATABASE_URL") == "" {
		if err := startTestPostgres(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  skipping testcontainer setup: %v\n", err)
		}
	}

	code := m.Run()
	shutdownTestPostgres()
	os.Exit(code)
}

func startTestPostgres() error {
	ctx := context.Background()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("could not start postgres container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
		return fmt.Errorf("failed to ping postgres: %w", err)
	}

	if err := applyTestSchema(db); err != nil {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
		return fmt.Errorf("failed to apply test schema: %w", err)
	}

	testContainer = pgContainer
	testDB = db
	os.Setenv("DATABASE_URL", connStr)
	os.Setenv("SCENARIO_NAME", "tidiness-manager-test")
	os.Setenv("POSTGRES_DB", "testdb")
	os.Setenv("POSTGRES_USER", "testuser")
	os.Setenv("POSTGRES_PASSWORD", "testpass")
	return nil
}

func shutdownTestPostgres() {
	ctx := context.Background()
	if testDB != nil {
		_ = testDB.Close()
	}
	if testContainer != nil {
		_ = testContainer.Terminate(ctx)
	}
}

// applyTestSchema creates the tables used by integration tests.
func applyTestSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS issues (
	id SERIAL PRIMARY KEY,
	scenario VARCHAR(255) NOT NULL,
	file_path TEXT NOT NULL,
	category VARCHAR(100) NOT NULL,
	severity VARCHAR(50) NOT NULL,
	title TEXT NOT NULL,
	description TEXT,
	line_number INTEGER,
	column_number INTEGER,
	agent_notes TEXT,
	remediation_steps TEXT,
	resolution_notes TEXT,
	status VARCHAR(50) DEFAULT 'open',
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	campaign_id INTEGER,
	session_id VARCHAR(255),
	resource_used TEXT
);

CREATE TABLE IF NOT EXISTS campaigns (
	id SERIAL PRIMARY KEY,
	scenario VARCHAR(255) NOT NULL,
	status VARCHAR(20) DEFAULT 'created',
	max_sessions INTEGER DEFAULT 10,
	max_files_per_session INTEGER DEFAULT 5,
	current_session INTEGER DEFAULT 0,
	files_visited INTEGER DEFAULT 0,
	files_total INTEGER DEFAULT 0,
	error_count INTEGER DEFAULT 0,
	error_reason TEXT,
	visited_tracker_campaign_id INTEGER,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS config (
	key VARCHAR(100) PRIMARY KEY,
	value TEXT NOT NULL,
	description TEXT,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS file_metrics (
	id SERIAL PRIMARY KEY,
	scenario VARCHAR(255) NOT NULL,
	file_path TEXT NOT NULL,
	language VARCHAR(50),
	file_extension VARCHAR(20),
	line_count INTEGER NOT NULL,
	todo_count INTEGER DEFAULT 0,
	fixme_count INTEGER DEFAULT 0,
	hack_count INTEGER DEFAULT 0,
	import_count INTEGER DEFAULT 0,
	function_count INTEGER DEFAULT 0,
	code_lines INTEGER DEFAULT 0,
	comment_lines INTEGER DEFAULT 0,
	comment_to_code_ratio DOUBLE PRECISION DEFAULT 0,
	has_test_file BOOLEAN DEFAULT FALSE,
	complexity_avg DOUBLE PRECISION,
	complexity_max INTEGER,
	duplication_pct DOUBLE PRECISION,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (scenario, file_path)
);
`

	_, err := db.Exec(schema)
	return err
}
