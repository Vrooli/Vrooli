package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	_ "github.com/lib/pq"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL        string
	Host       string
	Port       string
	User       string
	Password   string
	Database   string
	MaxRetries int
}

// ConnectWithRetry establishes a database connection with exponential backoff
func ConnectWithRetry(config DatabaseConfig) (*sql.DB, error) {
	dsn := config.URL

	if dsn == "" {
		// Fall back to individual components when URL is not provided
		if config.Host == "" {
			config.Host = "localhost"
		}
		if config.Port == "" {
			config.Port = "5433"
		}
		if config.User == "" {
			config.User = "vrooli"
		}
		if config.Database == "" {
			config.Database = "graph_studio"
		}

		if config.Password == "" {
			return nil, fmt.Errorf("database password is required when POSTGRES_URL is not set")
		}

		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			config.Host, config.Port, config.User, config.Password, config.Database)
	}

	var db *sql.DB
	var err error

	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 10
	}

	baseDelay := 500 * time.Millisecond
	maxDelay := 30 * time.Second
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔌 Attempting database connection (attempt %d/%d)...", attempt, maxRetries)

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Printf("❌ Failed to open database connection: %v", err)
			if attempt < maxRetries {
				delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt-1)), float64(maxDelay)))
				jitterRange := float64(delay) * 0.25
				jitter := time.Duration(randSource.Float64() * jitterRange)
				waitTime := delay + jitter
				log.Printf("⏳ Waiting %v before retry...", waitTime)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("failed to open database after %d attempts: %w", maxRetries, err)
		}

		// Configure connection pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		// Test the connection
		err = db.Ping()
		if err != nil {
			log.Printf("❌ Database ping failed: %v", err)
			db.Close()
			if attempt < maxRetries {
				delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt-1)), float64(maxDelay)))
				jitterRange := float64(delay) * 0.25
				jitter := time.Duration(randSource.Float64() * jitterRange)
				waitTime := delay + jitter
				log.Printf("⏳ Waiting %v before retry...", waitTime)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("database ping failed after %d attempts: %w", maxRetries, err)
		}

		log.Println("✅ Database connected successfully")
		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts", maxRetries)
}

// MonitorConnection monitors database connection health
// Note: sql.DB handles reconnection automatically via its connection pool
func MonitorConnection(db *sql.DB, config DatabaseConfig) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := db.Ping(); err != nil {
			log.Printf("⚠️  Database connection issue detected: %v", err)
			log.Println("🔄 Database pool will handle reconnection automatically")
		}
	}
}

// InitializePlugins inserts default plugins into the database
func InitializePlugins(db *sql.DB) error {
	// This is handled by the schema.sql file, but we keep this for runtime loading
	return nil
}
