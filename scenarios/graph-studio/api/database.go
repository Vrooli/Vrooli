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
	Host       string
	Port       string
	User       string
	Password   string
	Database   string
	MaxRetries int
}

// ConnectWithRetry establishes a database connection with exponential backoff
func ConnectWithRetry(config DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.Database)

	var db *sql.DB
	var err error

	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 10
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔌 Attempting database connection (attempt %d/%d)...", attempt, maxRetries)

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Printf("❌ Failed to open database connection: %v", err)
			if attempt < maxRetries {
				waitTime := calculateBackoff(attempt)
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
				waitTime := calculateBackoff(attempt)
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

// calculateBackoff calculates exponential backoff with jitter
func calculateBackoff(attempt int) time.Duration {
	// Base wait time with exponential increase
	base := time.Duration(math.Min(float64(attempt*attempt), 30)) * time.Second

	// Add random jitter (up to 25% of base time)
	jitter := time.Duration(float64(base) * 0.25 * rand.Float64())

	return base + jitter
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
