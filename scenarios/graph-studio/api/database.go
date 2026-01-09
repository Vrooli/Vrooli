package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
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
	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("✅ Database connected successfully")
	return db, nil
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
