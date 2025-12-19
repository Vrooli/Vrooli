package database

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

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

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
			return nil, fmt.Errorf("unsupported AM_DB_BACKEND: %s (expected postgres|sqlite)", dialect)
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
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
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
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
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
	return db.PingContext(ctx)
}

// WithTransaction executes a function within a database transaction.
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
		return fmt.Errorf("failed to read schema file at %s: %w", schemaPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultMigrationTimeout)
	defer cancel()

	_, err = db.ExecContext(ctx, string(schemaBytes))
	if err != nil {
		db.log.WithError(err).Error("Failed to execute schema initialization")
		return err
	}

	db.log.Info("Database schema initialized successfully")
	return nil
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
