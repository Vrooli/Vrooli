package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
)

type DB struct {
	*sqlx.DB
	log *logrus.Logger
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewConnection creates a new database connection with exponential backoff
func NewConnection(log *logrus.Logger) (*DB, error) {
	// Check for individual PostgreSQL environment variables (preferred)
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	// Override database name for this scenario if not specifically set
	if dbName == "vrooli" || dbName == "" {
		dbName = "browser_automation_studio"
	}

	var databaseURL string

	if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" && dbName != "" {
		// Construct URL from individual components
		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
		log.WithFields(logrus.Fields{
			"host":     dbHost,
			"port":     dbPort,
			"user":     dbUser,
			"database": dbName,
		}).Info("Using individual PostgreSQL environment variables")
	} else {
		// Fallback to DATABASE_URL if individual vars not available
		databaseURL = os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			return nil, fmt.Errorf("PostgreSQL configuration not found. Please set either DATABASE_URL or individual variables: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		log.Info("Using DATABASE_URL environment variable")
	}

	var db *sqlx.DB
	var err error

	// Exponential backoff configuration
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		db, err = sqlx.Connect("postgres", databaseURL)
		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseQueryTimeout)
			err = db.PingContext(ctx)
			cancel()

			if err == nil {
				log.Info("Successfully connected to database")
				break
			}
		}

		// Calculate delay with exponential backoff and random jitter
		delay := baseDelay * time.Duration(1<<attempt)
		if delay > maxDelay {
			delay = maxDelay
		}

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		log.WithFields(logrus.Fields{
			"attempt":    attempt + 1,
			"maxRetries": maxRetries,
			"delay":      actualDelay,
			"error":      err.Error(),
		}).Warn("Failed to connect to database, retrying...")

		time.Sleep(actualDelay)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	dbWrapper := &DB{
		DB:  db,
		log: log,
	}

	// Initialize database schema
	if err := dbWrapper.initSchema(); err != nil {
		log.WithError(err).Error("Failed to initialize database schema")
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return dbWrapper, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DatabasePingTimeout)
	defer cancel()

	return db.PingContext(ctx)
}

// WithTransaction executes a function within a database transaction
func (db *DB) WithTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// initSchema initializes the database schema
func (db *DB) initSchema() error {
	schema := `
-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    folder_path VARCHAR(500) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create workflow folders table
CREATE TABLE IF NOT EXISTS workflow_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path VARCHAR(500) NOT NULL UNIQUE,
    parent_id UUID REFERENCES workflow_folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create workflows table  
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    folder_path VARCHAR(500) NOT NULL,
    flow_definition JSONB DEFAULT '{}',
    description TEXT,
    tags TEXT[] DEFAULT '{}',
    version INTEGER DEFAULT 1,
    is_template BOOLEAN DEFAULT FALSE,
    created_by VARCHAR(255),
    last_change_source VARCHAR(255) DEFAULT 'manual',
    last_change_description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, folder_path)
);

-- Ensure newer workflow metadata columns exist for legacy databases
ALTER TABLE workflows
    ADD COLUMN IF NOT EXISTS last_change_source VARCHAR(255) DEFAULT 'manual';
ALTER TABLE workflows
    ALTER COLUMN last_change_source SET DEFAULT 'manual';
UPDATE workflows SET last_change_source = 'manual' WHERE last_change_source IS NULL;

ALTER TABLE workflows
    ADD COLUMN IF NOT EXISTS last_change_description TEXT;
ALTER TABLE workflows
    ALTER COLUMN last_change_description SET DEFAULT '';
UPDATE workflows SET last_change_description = '' WHERE last_change_description IS NULL;

ALTER TABLE workflows
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(255);

ALTER TABLE workflows
    ADD COLUMN IF NOT EXISTS project_id UUID;

-- Create workflow versions table
CREATE TABLE IF NOT EXISTS workflow_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    flow_definition JSONB NOT NULL,
    change_description TEXT,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version)
);

-- Create executions table
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    workflow_version INTEGER,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    trigger_type VARCHAR(50) NOT NULL DEFAULT 'manual',
    trigger_metadata JSONB DEFAULT '{}',
    parameters JSONB DEFAULT '{}',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    last_heartbeat TIMESTAMP,
    error TEXT,
    result JSONB,
    progress INTEGER DEFAULT 0,
    current_step VARCHAR(255)
);

-- Create execution logs table
CREATE TABLE IF NOT EXISTS execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    step_name VARCHAR(255),
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'
);

-- Create screenshots table
CREATE TABLE IF NOT EXISTS screenshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_name VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    storage_url VARCHAR(1000) NOT NULL,
    thumbnail_url VARCHAR(1000),
    width INTEGER,
    height INTEGER,
    size_bytes BIGINT,
    metadata JSONB DEFAULT '{}'
);

-- Create execution steps table
CREATE TABLE IF NOT EXISTS execution_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    step_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error TEXT,
    input JSONB DEFAULT '{}',
    output JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_execution_step UNIQUE (execution_id, step_index),
    CONSTRAINT chk_execution_step_status CHECK (status IN ('pending','running','completed','failed'))
);

-- Create execution artifacts table
CREATE TABLE IF NOT EXISTS execution_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_id UUID REFERENCES execution_steps(id) ON DELETE CASCADE,
    step_index INTEGER,
    artifact_type VARCHAR(100) NOT NULL,
    label VARCHAR(255),
    storage_url VARCHAR(1000),
    thumbnail_url VARCHAR(1000),
    content_type VARCHAR(100),
    size_bytes BIGINT,
    payload JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create extracted data table
CREATE TABLE IF NOT EXISTS extracted_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_name VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_key VARCHAR(255) NOT NULL,
    data_value JSONB NOT NULL,
    data_type VARCHAR(50),
    metadata JSONB DEFAULT '{}'
);

-- Create workflow schedules table
CREATE TABLE IF NOT EXISTS workflow_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    is_active BOOLEAN DEFAULT TRUE,
    parameters JSONB DEFAULT '{}',
    next_run_at TIMESTAMP,
    last_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create workflow templates table
CREATE TABLE IF NOT EXISTS workflow_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    flow_definition JSONB NOT NULL,
    icon VARCHAR(100),
    example_parameters JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create AI generations table
CREATE TABLE IF NOT EXISTS ai_generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    prompt TEXT NOT NULL,
    generated_flow JSONB NOT NULL,
    model VARCHAR(100),
    generation_time_ms INTEGER,
    success BOOLEAN DEFAULT FALSE,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_workflows_project_id ON workflows(project_id);
CREATE INDEX IF NOT EXISTS idx_workflows_folder_path ON workflows(folder_path);
CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_last_heartbeat ON executions(last_heartbeat);
CREATE INDEX IF NOT EXISTS idx_execution_logs_execution_id ON execution_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_screenshots_execution_id ON screenshots(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_steps_execution ON execution_steps(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_artifacts_execution ON execution_artifacts(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_artifacts_step ON execution_artifacts(step_id);
CREATE INDEX IF NOT EXISTS idx_extracted_data_execution_id ON extracted_data(execution_id);
CREATE INDEX IF NOT EXISTS idx_workflow_schedules_active ON workflow_schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_ai_generations_workflow_id ON ai_generations(workflow_id);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflow_folders_updated_at ON workflow_folders;
CREATE TRIGGER update_workflow_folders_updated_at BEFORE UPDATE ON workflow_folders FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflow_schedules_updated_at ON workflow_schedules;
CREATE TRIGGER update_workflow_schedules_updated_at BEFORE UPDATE ON workflow_schedules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflow_templates_updated_at ON workflow_templates;
CREATE TRIGGER update_workflow_templates_updated_at BEFORE UPDATE ON workflow_templates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_execution_steps_updated_at ON execution_steps;
CREATE TRIGGER update_execution_steps_updated_at BEFORE UPDATE ON execution_steps FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_execution_artifacts_updated_at ON execution_artifacts;
CREATE TRIGGER update_execution_artifacts_updated_at BEFORE UPDATE ON execution_artifacts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
	`

	ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseMigrationTimeout)
	defer cancel()

	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		db.log.WithError(err).Error("Failed to execute schema initialization")
		return err
	}

	db.log.Info("Database schema initialized successfully")

	// Skip demo seeding in test environments
	if os.Getenv("BAS_SKIP_DEMO_SEED") != "true" {
		if err := db.seedDemoWorkflow(); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) seedDemoWorkflow() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const demoWorkflowFolder = "/demo"
	const demoProjectName = "Demo Browser Automations"
	const demoProjectDescription = "Seeded workflows that showcase Browser Automation Studio execution, telemetry, and replay features."
	const demoWorkflowName = "Demo: Capture Example.com Hero"

	demoProjectFolder := os.Getenv("BAS_DEMO_PROJECT_PATH")
	if demoProjectFolder == "" {
		demoProjectFolder = filepath.Join("scenarios", "browser-automation-studio", "data", "projects", "demo")
	}

	absDemoProjectFolder, err := filepath.Abs(demoProjectFolder)
	if err != nil {
		return fmt.Errorf("failed to resolve demo project folder: %w", err)
	}

	if err := os.MkdirAll(absDemoProjectFolder, 0o755); err != nil {
		return fmt.Errorf("failed to create demo project directory: %w", err)
	}

	type projectRow struct {
		ID         uuid.UUID `db:"id"`
		FolderPath string    `db:"folder_path"`
	}

	var project projectRow
	err = db.GetContext(ctx, &project, `SELECT id, folder_path FROM projects WHERE name = $1 LIMIT 1`, demoProjectName)
	switch {
	case err == nil:
		if filepath.Clean(project.FolderPath) != filepath.Clean(absDemoProjectFolder) {
			if _, err := db.ExecContext(
				ctx,
				`UPDATE projects SET folder_path = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
				absDemoProjectFolder,
				project.ID,
			); err != nil {
				return fmt.Errorf("failed to update demo project folder: %w", err)
			}
			project.FolderPath = absDemoProjectFolder
		}
	case err == sql.ErrNoRows:
		project = projectRow{ID: uuid.New(), FolderPath: absDemoProjectFolder}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO projects (id, name, description, folder_path, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			 ON CONFLICT (folder_path) DO NOTHING`,
			project.ID,
			demoProjectName,
			demoProjectDescription,
			absDemoProjectFolder,
		); err != nil {
			return fmt.Errorf("failed to seed demo project: %w", err)
		}

		if err := db.GetContext(ctx, &project, `SELECT id, folder_path FROM projects WHERE name = $1 LIMIT 1`, demoProjectName); err != nil {
			return fmt.Errorf("failed to confirm demo project id: %w", err)
		}

		if db.log != nil {
			db.log.WithFields(logrus.Fields{
				"project_id":  project.ID,
				"folder_path": project.FolderPath,
			}).Info("Seeded demo project 'Demo Browser Automations'")
		}
	default:
		return fmt.Errorf("failed to lookup demo project: %w", err)
	}

	folderPath := demoWorkflowFolder
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO workflow_folders (id, path, name, description, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT (path) DO NOTHING`,
		uuid.New(),
		folderPath,
		"Demo Workflows",
		"Sample browser automation journeys seeded by Browser Automation Studio",
	); err != nil {
		return fmt.Errorf("failed to seed demo workflow folder: %w", err)
	}

	flowDefinition := JSONMap{
		"nodes": []any{
			map[string]any{
				"id":   "navigate-home",
				"type": "navigate",
				"position": map[string]any{
					"x": -320,
					"y": -60,
				},
				"data": map[string]any{
					"label":     "Navigate to Example",
					"url":       "https://example.com",
					"waitUntil": "networkidle2",
					"timeoutMs": 20000,
				},
			},
			map[string]any{
				"id":   "wait-for-hero",
				"type": "wait",
				"position": map[string]any{
					"x": -80,
					"y": -60,
				},
				"data": map[string]any{
					"label":    "Allow page to settle",
					"type":     "time",
					"duration": 1500,
				},
			},
			map[string]any{
				"id":   "assert-hero",
				"type": "assert",
				"position": map[string]any{
					"x": 160,
					"y": -60,
				},
				"data": map[string]any{
					"label":             "Verify hero heading",
					"mode":              "text_contains",
					"selector":          "h1",
					"expectedValue":     "Example Domain",
					"continueOnFailure": false,
					"caseSensitive":     false,
				},
			},
			map[string]any{
				"id":   "screenshot-hero",
				"type": "screenshot",
				"position": map[string]any{
					"x": 400,
					"y": -60,
				},
				"data": map[string]any{
					"label":              "Capture hero",
					"name":               "example-hero",
					"focusSelector":      "h1",
					"highlightSelectors": []any{"h1", "p"},
					"highlightColor":     "#38bdf8",
					"highlightPadding":   12,
					"maskSelectors":      []any{"a"},
					"maskOpacity":        0.5,
					"zoomFactor":         1.15,
					"background":         "#0f172a",
					"waitForMs":          500,
				},
			},
		},
		"edges": []any{
			map[string]any{"id": "edge-navigate-wait", "source": "navigate-home", "target": "wait-for-hero"},
			map[string]any{"id": "edge-wait-assert", "source": "wait-for-hero", "target": "assert-hero"},
			map[string]any{"id": "edge-assert-screenshot", "source": "assert-hero", "target": "screenshot-hero"},
		},
	}

	type workflowRow struct {
		ID         uuid.UUID      `db:"id"`
		ProjectID  *uuid.UUID     `db:"project_id"`
		FolderPath string         `db:"folder_path"`
		CreatedBy  sql.NullString `db:"created_by"`
	}

	var workflow workflowRow
	err = db.GetContext(ctx, &workflow, `SELECT id, project_id, folder_path FROM workflows WHERE name = $1 LIMIT 1`, demoWorkflowName)
	switch {
	case err == nil:
		needsUpdate := false
		if workflow.ProjectID == nil || *workflow.ProjectID != project.ID {
			needsUpdate = true
		}
		if filepath.Clean(workflow.FolderPath) != filepath.Clean(folderPath) {
			needsUpdate = true
		}
		createdByValue := "seed"
		if workflow.CreatedBy.Valid {
			createdByValue = workflow.CreatedBy.String
		}
		if !workflow.CreatedBy.Valid {
			needsUpdate = true
		}
		if needsUpdate {
			if _, err := db.ExecContext(
				ctx,
				`UPDATE workflows SET project_id = $1, folder_path = $2, created_by = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4`,
				project.ID,
				folderPath,
				createdByValue,
				workflow.ID,
			); err != nil {
				return fmt.Errorf("failed to update demo workflow metadata: %w", err)
			}
		}
		return nil
	case err != sql.ErrNoRows:
		return fmt.Errorf("failed to lookup demo workflow: %w", err)
	}

	workflow.ID = uuid.New()
	description := "Demo workflow that loads example.com, verifies the hero heading, and captures an annotated screenshot."

	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO workflows (id, project_id, name, folder_path, flow_definition, description, version, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 1, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		workflow.ID,
		project.ID,
		demoWorkflowName,
		folderPath,
		flowDefinition,
		description,
		"seed",
	); err != nil {
		return fmt.Errorf("failed to seed demo workflow: %w", err)
	}

	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO workflow_versions (id, workflow_id, version, flow_definition, change_description, created_by, created_at)
		 VALUES ($1, $2, 1, $3, $4, $5, CURRENT_TIMESTAMP)
		 ON CONFLICT (workflow_id, version) DO NOTHING`,
		uuid.New(),
		workflow.ID,
		flowDefinition,
		"Initial demo workflow seed",
		"seed",
	); err != nil {
		return fmt.Errorf("failed to seed demo workflow version: %w", err)
	}

	if db.log != nil {
		db.log.WithFields(logrus.Fields{
			"workflow_id": workflow.ID,
			"project_id":  project.ID,
		}).Info("Seeded demo workflow 'Demo: Capture Example.com Hero'")
	}

	return nil
}
