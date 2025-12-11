package database

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/constants"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sqlx.DB
	log     *logrus.Logger
	backend string
	dialect Dialect
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewConnection creates a new database connection with exponential backoff
func NewConnection(log *logrus.Logger) (*DB, error) {
	backend := parseDialect(os.Getenv("BAS_DB_BACKEND"))

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
	var db *sqlx.DB
	var err error

	// Load configuration from control surface
	cfg := config.Load()

	// Exponential backoff configuration from control surface
	maxRetries := cfg.Database.MaxRetries
	baseDelay := cfg.Database.BaseRetryDelay
	maxDelay := cfg.Database.MaxRetryDelay
	jitterFactor := cfg.Database.RetryJitterFactor

	for attempt := 0; attempt < maxRetries; attempt++ {
		switch backend {
		case DialectPostgres:
			if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" && dbName != "" {
				databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
					dbUser, dbPassword, dbHost, dbPort, dbName)
				log.WithFields(logrus.Fields{
					"host":     dbHost,
					"port":     dbPort,
					"user":     dbUser,
					"database": dbName,
				}).Info("Using individual PostgreSQL environment variables")
			} else {
				databaseURL = os.Getenv("DATABASE_URL")
				if databaseURL == "" {
					return nil, fmt.Errorf("PostgreSQL configuration not found. Please set either DATABASE_URL or individual variables: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
				}
				log.Info("Using DATABASE_URL environment variable")
			}
			db, err = sqlx.Connect("postgres", databaseURL)
		case DialectSQLite:
			databaseURL, err = sqliteDSN(log)
			if err != nil {
				return nil, err
			}
			if err := ensureSQLiteResource(log); err != nil {
				log.WithError(err).Warn("SQLite resource install check failed; continuing with direct driver")
			}
			db, err = sqlx.Connect("sqlite", databaseURL)
		default:
			return nil, fmt.Errorf("unsupported BAS_DB_BACKEND: %s (expected postgres|sqlite)", backend)
		}

		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseQueryTimeout)
			err = db.PingContext(ctx)
			cancel()

			if err == nil {
				log.WithField("backend", backend).Info("Successfully connected to database")
				break
			}
		}

		// Calculate delay with exponential backoff and random jitter
		delay := baseDelay * time.Duration(1<<attempt)
		if delay > maxDelay {
			delay = maxDelay
		}

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * jitterFactor
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

	// Configure connection pool from control surface
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	if backend == "sqlite" {
		// SQLite only supports a single connection
		db.SetMaxOpenConns(1)
	}

	dbWrapper := &DB{
		DB:      db,
		log:     log,
		backend: string(backend),
		dialect: backend,
	}
	currentDialect = backend // legacy global for types still reading it
	SetDialectProvider(dbWrapper)

	// Initialize database schema
	if err := dbWrapper.initSchema(); err != nil {
		log.WithError(err).Error("Failed to initialize database schema")
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return dbWrapper, nil
}

// Dialect returns the active dialect for this DB.
func (db *DB) Dialect() Dialect {
	if db == nil {
		return DialectPostgres
	}
	return db.dialect
}

// RawDB returns the underlying *sql.DB for direct database access.
// Used by services that need raw SQL execution (e.g., entitlement usage tracking).
func (db *DB) RawDB() *sql.DB {
	return db.DB.DB
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

// initSchema initializes the database schema by loading and executing the schema file.
// The schema file is loaded from disk at runtime and contains all table definitions,
// indexes, and triggers required by the browser automation platform.
func (db *DB) initSchema() error {
	// Determine the schema file path relative to this source file
	filename := "schema.sql"
	if db.backend == "sqlite" {
		filename = "schema_sqlite.sql"
	}
	schemaPath := filepath.Join(filepath.Dir(getCurrentFilePath()), filename)

	// Read the schema file
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		db.log.WithError(err).WithField("path", schemaPath).Error("Failed to read schema file")
		return fmt.Errorf("failed to read schema file at %s: %w", schemaPath, err)
	}

	schema := string(schemaBytes)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseMigrationTimeout)
	defer cancel()

	_, err = db.ExecContext(ctx, schema)
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

// getCurrentFilePath returns the path of the current source file.
// This is used to locate the schema file relative to connection.go.
func getCurrentFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filename
}

func sqliteDSN(log *logrus.Logger) (string, error) {
	root := strings.TrimSpace(os.Getenv("BAS_SQLITE_PATH"))
	if root == "" {
		if custom := strings.TrimSpace(os.Getenv("DATABASE_URL")); strings.HasPrefix(custom, "file:") {
			// Allow explicit file URLs (desktop bundles can set this).
			return custom, nil
		}
		dataRoot := strings.TrimSpace(os.Getenv("SQLITE_DATABASE_PATH"))
		if dataRoot == "" {
			dataRoot = strings.TrimSpace(os.Getenv("VROOLI_DATA"))
		}
		if dataRoot == "" {
			home, _ := os.UserHomeDir()
			if home == "" {
				home = "."
			}
			dataRoot = filepath.Join(home, ".vrooli", "data", "sqlite", "databases")
		}
		root = filepath.Join(dataRoot, "browser-automation-studio.db")
	}
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}

	// Mirror resources/sqlite defaults; allow env override if caller provides a full DATABASE_URL=file:...
	if log != nil {
		log.WithFields(logrus.Fields{
			"path":    root,
			"backend": "sqlite",
		}).Info("Using SQLite DSN derived from resource defaults")
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(268435456)",
		root,
	), nil
}

// ensureSQLiteResource best-effort invokes the sqlite resource install to align directories/permissions.
func ensureSQLiteResource(log *logrus.Logger) error {
	cli := strings.TrimSpace(os.Getenv("SQLITE_CLI_PATH"))
	if cli == "" {
		// Prefer the resource-local shim if present.
		cli = filepath.Join("resources", "sqlite", "cli.sh")
		if _, err := os.Stat(cli); err != nil {
			cli = "resource-sqlite"
		}
	}
	cmd := exec.Command(cli, "manage", "install")
	// Run quietly; we only care about success/failure.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if log != nil {
			log.WithError(err).Debug("resource-sqlite manage install failed or missing")
		}
		return err
	}
	return nil
}

func (db *DB) seedDemoWorkflow() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const demoWorkflowFolder = "/demo"
	const demoProjectName = "Demo Browser Automations"
	const demoProjectDescription = "Seeded workflows that showcase Vrooli Ascension execution, telemetry, and replay features."
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

	selectProject := db.Rebind(`SELECT id, folder_path FROM projects WHERE folder_path = ? LIMIT 1`)
	var project projectRow
	err = db.GetContext(ctx, &project, selectProject, absDemoProjectFolder)
	switch {
	case err == nil:
		// Project already exists with this folder_path, use it
	case err == sql.ErrNoRows:
		project = projectRow{ID: uuid.New(), FolderPath: absDemoProjectFolder}
		insertProject := db.Rebind(
			`INSERT INTO projects (id, name, description, folder_path, created_at, updated_at)
			 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			 ON CONFLICT (folder_path) DO NOTHING`)
		if _, err := db.ExecContext(
			ctx,
			insertProject,
			project.ID,
			demoProjectName,
			demoProjectDescription,
			absDemoProjectFolder,
		); err != nil {
			return fmt.Errorf("failed to seed demo project: %w", err)
		}

		if err := db.GetContext(ctx, &project, selectProject, absDemoProjectFolder); err != nil {
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
	insertFolder := db.Rebind(
		`INSERT INTO workflow_folders (id, path, name, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT (path) DO NOTHING`)
	if _, err := db.ExecContext(
		ctx,
		insertFolder,
		uuid.New(),
		folderPath,
		"Demo Workflows",
		"Sample browser automation journeys seeded by Vrooli Ascension",
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
					"waitUntil": "networkidle",
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
	selectWorkflow := db.Rebind(`SELECT id, project_id, folder_path FROM workflows WHERE name = ? LIMIT 1`)
	err = db.GetContext(ctx, &workflow, selectWorkflow, demoWorkflowName)
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
			updateWorkflow := db.Rebind(`UPDATE workflows SET project_id = ?, folder_path = ?, created_by = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
			if _, err := db.ExecContext(ctx, updateWorkflow, project.ID, folderPath, createdByValue, workflow.ID); err != nil {
				return fmt.Errorf("failed to update demo workflow metadata: %w", err)
			}
		}
		return nil
	case err != sql.ErrNoRows:
		return fmt.Errorf("failed to lookup demo workflow: %w", err)
	}

	workflow.ID = uuid.New()
	description := "Demo workflow that loads example.com, verifies the hero heading, and captures an annotated screenshot."

	insertWorkflow := db.Rebind(`INSERT INTO workflows (id, project_id, name, folder_path, flow_definition, description, version, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 1, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if _, err := db.ExecContext(ctx, insertWorkflow, workflow.ID, project.ID, demoWorkflowName, folderPath, flowDefinition, description, "seed"); err != nil {
		return fmt.Errorf("failed to seed demo workflow: %w", err)
	}

	insertVersion := db.Rebind(`INSERT INTO workflow_versions (id, workflow_id, version, flow_definition, change_description, created_by, created_at)
		 VALUES (?, ?, 1, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT (workflow_id, version) DO NOTHING`)
	if _, err := db.ExecContext(ctx, insertVersion, uuid.New(), workflow.ID, flowDefinition, "Initial demo workflow seed", "seed"); err != nil {
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
