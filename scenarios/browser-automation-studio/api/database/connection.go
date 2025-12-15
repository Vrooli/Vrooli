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

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/constants"
	_ "modernc.org/sqlite"
)

// DB wraps sqlx.DB with additional functionality
type DB struct {
	*sqlx.DB
	log     *logrus.Logger
	dialect Dialect
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewConnection creates a new database connection with exponential backoff
func NewConnection(log *logrus.Logger) (*DB, error) {
	dialect := parseDialect(os.Getenv("BAS_DB_BACKEND"))

	var db *sqlx.DB
	var err error

	// Load configuration from control surface
	cfg := config.Load()

	// Exponential backoff configuration
	maxRetries := cfg.Database.MaxRetries
	baseDelay := cfg.Database.BaseRetryDelay
	maxDelay := cfg.Database.MaxRetryDelay
	jitterFactor := cfg.Database.RetryJitterFactor

	for attempt := 0; attempt < maxRetries; attempt++ {
		switch dialect {
		case DialectPostgres:
			db, err = connectPostgres(log)
		case DialectSQLite:
			db, err = connectSQLite(log)
		default:
			return nil, fmt.Errorf("unsupported BAS_DB_BACKEND: %s (expected postgres|sqlite)", dialect)
		}

		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseQueryTimeout)
			err = db.PingContext(ctx)
			cancel()

			if err == nil {
				log.WithField("dialect", dialect).Info("Successfully connected to database")
				break
			}
		}

		// Calculate delay with exponential backoff and random jitter
		delay := baseDelay * time.Duration(1<<attempt)
		if delay > maxDelay {
			delay = maxDelay
		}
		jitter := time.Duration(float64(delay) * jitterFactor * rand.Float64())
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
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	if dialect == DialectSQLite {
		db.SetMaxOpenConns(1) // SQLite only supports a single connection
	}

	dbWrapper := &DB{
		DB:      db,
		log:     log,
		dialect: dialect,
	}

	// Initialize database schema
	if err := dbWrapper.initSchema(); err != nil {
		log.WithError(err).Error("Failed to initialize database schema")
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return dbWrapper, nil
}

func connectPostgres(log *logrus.Logger) (*sqlx.DB, error) {
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
			return nil, fmt.Errorf("PostgreSQL configuration not found. Set DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB")
		}
		log.Info("Using DATABASE_URL environment variable")
	}

	return sqlx.Connect("postgres", databaseURL)
}

func connectSQLite(log *logrus.Logger) (*sqlx.DB, error) {
	dsn, err := sqliteDSN(log)
	if err != nil {
		return nil, err
	}

	// Best-effort invoke sqlite resource install
	if err := ensureSQLiteResource(log); err != nil {
		log.WithError(err).Debug("SQLite resource install check failed; continuing")
	}

	return sqlx.Connect("sqlite", dsn)
}

func sqliteDSN(log *logrus.Logger) (string, error) {
	root := strings.TrimSpace(os.Getenv("BAS_SQLITE_PATH"))
	if root == "" {
		if custom := strings.TrimSpace(os.Getenv("DATABASE_URL")); strings.HasPrefix(custom, "file:") {
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

	if log != nil {
		log.WithField("path", root).Info("Using SQLite database")
	}

	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(268435456)",
		root,
	), nil
}

func ensureSQLiteResource(log *logrus.Logger) error {
	cli := strings.TrimSpace(os.Getenv("SQLITE_CLI_PATH"))
	if cli == "" {
		cli = filepath.Join("resources", "sqlite", "cli.sh")
		if _, err := os.Stat(cli); err != nil {
			cli = "resource-sqlite"
		}
	}
	cmd := exec.Command(cli, "manage", "install")
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

// Dialect returns the active dialect for this DB
func (db *DB) Dialect() Dialect {
	if db == nil {
		return DialectPostgres
	}
	return db.dialect
}

// RawDB returns the underlying *sql.DB for direct database access
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

// initSchema initializes the database schema
func (db *DB) initSchema() error {
	filename := "schema.sql"
	if db.dialect == DialectSQLite {
		filename = "schema_sqlite.sql"
	}
	schemaPath := filepath.Join(filepath.Dir(getCurrentFilePath()), filename)

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		db.log.WithError(err).WithField("path", schemaPath).Error("Failed to read schema file")
		return fmt.Errorf("failed to read schema file at %s: %w", schemaPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseMigrationTimeout)
	defer cancel()

	_, err = db.ExecContext(ctx, string(schemaBytes))
	if err != nil {
		db.log.WithError(err).Error("Failed to execute schema initialization")
		return err
	}

	if err := db.applyIndexSchemaMigrations(ctx); err != nil {
		db.log.WithError(err).Error("Failed to apply index schema migrations")
		return err
	}

	db.log.Info("Database schema initialized successfully")
	return nil
}

func (db *DB) applyIndexSchemaMigrations(ctx context.Context) error {
	if db.dialect != DialectPostgres {
		return nil
	}

	// Legacy database instances may already have tables created without newer index columns.
	// Avoid SELECT * scan breakages by ensuring expected columns exist.
	statements := []string{
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS folder_path VARCHAR(500)`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,

		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS project_id UUID`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS folder_path VARCHAR(500)`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS file_path VARCHAR(1000)`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,

		`ALTER TABLE executions ADD COLUMN IF NOT EXISTS status VARCHAR(50)`,
		`ALTER TABLE executions ADD COLUMN IF NOT EXISTS started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE executions ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP`,
		`ALTER TABLE executions ADD COLUMN IF NOT EXISTS error_message TEXT`,
		`ALTER TABLE executions ADD COLUMN IF NOT EXISTS result_path VARCHAR(1000)`,
		`ALTER TABLE executions ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE executions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,

		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS name VARCHAR(255)`,
		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS cron_expression VARCHAR(100)`,
		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'UTC'`,
		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE`,
		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS parameters_json TEXT DEFAULT '{}'`,
		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMP`,
		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS last_run_at TIMESTAMP`,
		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE schedules ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("schema migration failed: %w", err)
		}
	}

	return nil
}

func getCurrentFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filename
}
