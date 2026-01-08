package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // SQLite driver
)

// Default configuration values
const (
	defaultMaxOpenConns      = 25
	defaultMaxIdleConns      = 5
	defaultConnMaxLifetimeMs = 300000 // 5 minutes
	defaultMaxRetries        = 10
	defaultBaseRetryDelayMs  = 1000  // 1 second
	defaultMaxRetryDelayMs   = 30000 // 30 seconds
	defaultRetryJitterFactor = 0.25
	defaultQueryTimeout      = 30 * time.Second
	defaultMigrationTimeout  = 60 * time.Second
	defaultPingTimeout       = 5 * time.Second
)

// DB wraps sqlx.DB with additional functionality for agent-manager.
type DB struct {
	*sqlx.DB
	log     *logrus.Logger
	dialect Dialect
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// NewConnection creates a new database connection with exponential backoff retry.
func NewConnection(log *logrus.Logger) (*DB, error) {
	dialect := getDialectFromEnv()

	// Load configuration from environment
	maxRetries := getEnvInt("AM_DB_MAX_RETRIES", defaultMaxRetries)
	baseDelay := time.Duration(getEnvInt("AM_DB_BASE_RETRY_DELAY_MS", defaultBaseRetryDelayMs)) * time.Millisecond
	maxDelay := time.Duration(getEnvInt("AM_DB_MAX_RETRY_DELAY_MS", defaultMaxRetryDelayMs)) * time.Millisecond
	jitterFactor := getEnvFloat("AM_DB_RETRY_JITTER_FACTOR", defaultRetryJitterFactor)

	var db *sqlx.DB
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		switch dialect {
		case DialectPostgres:
			db, err = connectPostgres(log)
		case DialectSQLite:
			db, err = connectSQLite(log)
		default:
			return nil, domain.NewConfigInvalidError("AM_DB_BACKEND", fmt.Sprintf("unsupported value: %s (expected postgres|sqlite)", dialect), nil)
		}

		if err != nil {
			if domain.AsDomainError(err) != nil {
				return nil, err
			}
		}

		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
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
		jitter := time.Duration(float64(delay) * jitterFactor * rng.Float64())
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
		return nil, &domain.DatabaseError{
			Operation:   "connect",
			EntityType:  "Database",
			Cause:       err,
			IsTransient: true,
		}
	}

	// Configure connection pool
	maxOpenConns := getEnvInt("AM_DB_MAX_OPEN_CONNS", defaultMaxOpenConns)
	maxIdleConns := getEnvInt("AM_DB_MAX_IDLE_CONNS", defaultMaxIdleConns)
	connMaxLifetime := time.Duration(getEnvInt("AM_DB_CONN_MAX_LIFETIME_MS", defaultConnMaxLifetimeMs)) * time.Millisecond

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// SQLite only supports a single connection
	if dialect == DialectSQLite {
		db.SetMaxOpenConns(1)
	}

	dbWrapper := &DB{
		DB:      db,
		log:     log,
		dialect: dialect,
	}

	// Initialize database schema
	if err := dbWrapper.initSchema(); err != nil {
		log.WithError(err).Error("Failed to initialize database schema")
		return nil, err
	}

	return dbWrapper, nil
}

func connectPostgres(log *logrus.Logger) (*sqlx.DB, error) {
	// Check for individual PostgreSQL environment variables
	dbHost := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	dbPort := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	dbUser := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	dbPassword := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	dbName := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	// Override database name for this scenario if not specifically set
	if dbName == "vrooli" || dbName == "" {
		dbName = "agent_manager"
	}

	var databaseURL string
	if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" {
		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
		log.WithFields(logrus.Fields{
			"host":     dbHost,
			"port":     dbPort,
			"user":     dbUser,
			"database": dbName,
		}).Info("Using individual PostgreSQL environment variables")
	} else {
		databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
		if databaseURL == "" {
			return nil, domain.NewConfigMissingError("DATABASE_URL", "PostgreSQL configuration not found. Set DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB", nil)
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
	return sqlx.Connect("sqlite", dsn)
}

func sqliteDSN(log *logrus.Logger) (string, error) {
	root := strings.TrimSpace(os.Getenv("AM_SQLITE_PATH"))
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
		root = filepath.Join(dataRoot, "agent-manager.db")
	}

	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return "", domain.NewConfigInvalidError("AM_SQLITE_PATH", "prepare sqlite directory", err)
	}

	if log != nil {
		log.WithField("path", root).Info("Using SQLite database")
	}

	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		root,
	), nil
}

// Dialect returns the active dialect for this DB.
func (db *DB) Dialect() Dialect {
	if db == nil {
		return DialectPostgres
	}
	return db.dialect
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck performs a health check on the database.
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return &domain.DatabaseError{
			Operation:   "health_check",
			EntityType:  "Database",
			Cause:       err,
			IsTransient: true,
		}
	}
	return nil
}

// WithTransaction executes a function within a database transaction.
func (db *DB) WithTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return &domain.DatabaseError{
			Operation:   "transaction_begin",
			EntityType:  "Database",
			Cause:       err,
			IsTransient: true,
		}
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return &domain.DatabaseError{
			Operation:   "transaction_commit",
			EntityType:  "Database",
			Cause:       err,
			IsTransient: true,
		}
	}
	return nil
}

// initSchema initializes the database schema.
func (db *DB) initSchema() error {
	filename := "schema.sql"
	if db.dialect == DialectSQLite {
		filename = "schema_sqlite.sql"
	}
	schemaPath := filepath.Join(filepath.Dir(getCurrentFilePath()), filename)

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		db.log.WithError(err).WithField("path", schemaPath).Error("Failed to read schema file")
		return &domain.DatabaseError{
			Operation:  "schema_read",
			EntityType: "Schema",
			Cause:      fmt.Errorf("failed to read schema file at %s: %w", schemaPath, err),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultMigrationTimeout)
	defer cancel()

	_, err = db.ExecContext(ctx, string(schemaBytes))
	if err != nil {
		db.log.WithError(err).Error("Failed to execute schema initialization")
		return &domain.DatabaseError{
			Operation:  "schema_init",
			EntityType: "Schema",
			Cause:      err,
		}
	}

	if err := db.ensureProfileKeyColumn(ctx); err != nil {
		db.log.WithError(err).Error("Failed to backfill profile keys")
		return err
	}
	if err := db.ensureModelPresetColumn(ctx); err != nil {
		db.log.WithError(err).Error("Failed to backfill model preset column")
		return err
	}
	if err := db.ensureFallbackRunnerTypesColumn(ctx); err != nil {
		db.log.WithError(err).Error("Failed to backfill fallback runner types column")
		return err
	}

	db.log.Info("Database schema initialized successfully")
	return nil
}

func (db *DB) ensureProfileKeyColumn(ctx context.Context) error {
	switch db.dialect {
	case DialectPostgres:
		var exists bool
		if err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'agent_profiles'
				AND column_name = 'profile_key'
			)`).Scan(&exists); err != nil {
			return schemaMigrationError("profile_key_check", err)
		}
		if !exists {
			if _, err := db.ExecContext(ctx, `ALTER TABLE agent_profiles ADD COLUMN profile_key VARCHAR(255)`); err != nil {
				return schemaMigrationError("profile_key_add", err)
			}
		}

		if _, err := db.ExecContext(ctx, `UPDATE agent_profiles SET profile_key = name WHERE profile_key IS NULL OR profile_key = ''`); err != nil {
			return schemaMigrationError("profile_key_backfill", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE agent_profiles ALTER COLUMN profile_key SET NOT NULL`); err != nil {
			return schemaMigrationError("profile_key_not_null", err)
		}

		if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_profiles_profile_key ON agent_profiles(profile_key)`); err != nil {
			return schemaMigrationError("profile_key_index", err)
		}
	case DialectSQLite:
		exists, err := sqliteColumnExists(ctx, db, "agent_profiles", "profile_key")
		if err != nil {
			return schemaMigrationError("profile_key_check", err)
		}
		if !exists {
			if _, err := db.ExecContext(ctx, `ALTER TABLE agent_profiles ADD COLUMN profile_key TEXT`); err != nil {
				return schemaMigrationError("profile_key_add", err)
			}
		}
		if _, err := db.ExecContext(ctx, `UPDATE agent_profiles SET profile_key = name WHERE profile_key IS NULL OR profile_key = ''`); err != nil {
			return schemaMigrationError("profile_key_backfill", err)
		}
		if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_profiles_profile_key ON agent_profiles(profile_key)`); err != nil {
			return schemaMigrationError("profile_key_index", err)
		}
	default:
		return domain.NewConfigInvalidError("AM_DB_BACKEND", fmt.Sprintf("unsupported dialect: %s", db.dialect), nil)
	}

	return nil
}

func (db *DB) ensureModelPresetColumn(ctx context.Context) error {
	switch db.dialect {
	case DialectPostgres:
		var exists bool
		if err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'agent_profiles'
				AND column_name = 'model_preset'
			)`).Scan(&exists); err != nil {
			return schemaMigrationError("model_preset_check", err)
		}
		if !exists {
			if _, err := db.ExecContext(ctx, `ALTER TABLE agent_profiles ADD COLUMN model_preset VARCHAR(20)`); err != nil {
				return schemaMigrationError("model_preset_add", err)
			}
		}
	case DialectSQLite:
		exists, err := sqliteColumnExists(ctx, db, "agent_profiles", "model_preset")
		if err != nil {
			return schemaMigrationError("model_preset_check", err)
		}
		if !exists {
			if _, err := db.ExecContext(ctx, `ALTER TABLE agent_profiles ADD COLUMN model_preset TEXT`); err != nil {
				return schemaMigrationError("model_preset_add", err)
			}
		}
	default:
		return domain.NewConfigInvalidError("AM_DB_BACKEND", fmt.Sprintf("unsupported dialect: %s", db.dialect), nil)
	}
	return nil
}

func (db *DB) ensureFallbackRunnerTypesColumn(ctx context.Context) error {
	switch db.dialect {
	case DialectPostgres:
		var exists bool
		if err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'agent_profiles'
				AND column_name = 'fallback_runner_types'
			)`).Scan(&exists); err != nil {
			return schemaMigrationError("fallback_runner_types_check", err)
		}
		if !exists {
			if _, err := db.ExecContext(ctx, `ALTER TABLE agent_profiles ADD COLUMN fallback_runner_types JSONB DEFAULT '[]'`); err != nil {
				return schemaMigrationError("fallback_runner_types_add", err)
			}
		}
	case DialectSQLite:
		exists, err := sqliteColumnExists(ctx, db, "agent_profiles", "fallback_runner_types")
		if err != nil {
			return schemaMigrationError("fallback_runner_types_check", err)
		}
		if !exists {
			if _, err := db.ExecContext(ctx, `ALTER TABLE agent_profiles ADD COLUMN fallback_runner_types TEXT DEFAULT '[]'`); err != nil {
				return schemaMigrationError("fallback_runner_types_add", err)
			}
		}
	default:
		return domain.NewConfigInvalidError("AM_DB_BACKEND", fmt.Sprintf("unsupported dialect: %s", db.dialect), nil)
	}
	return nil
}

func schemaMigrationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return &domain.DatabaseError{
		Operation:  operation,
		EntityType: "Schema",
		Cause:      err,
	}
}

func sqliteColumnExists(ctx context.Context, db *DB, tableName, columnName string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid      int
			name     string
			colType  string
			notNull  int
			defaultV sql.NullString
			primaryK int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primaryK); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, rows.Err()
}

func getCurrentFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filename
}

// Helper functions for reading environment variables
func getEnvInt(key string, defaultVal int) int {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}
