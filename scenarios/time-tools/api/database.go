package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
)

var db *sql.DB

// initDB initializes the PostgreSQL database connection with automatic retry and backoff.
// Reads POSTGRES_* environment variables set by the lifecycle system.
// Database is optional for time-tools - it can work without it.
func initDB(logger *log.Logger) error {
	// Check if database configuration is available
	if os.Getenv("POSTGRES_HOST") == "" && os.Getenv("POSTGRES_PORT") == "" {
		logger.Println("Database configuration not found, running without database")
		return nil
	}

	var err error
	db, err = database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		// Database connection failed, but we can still run without it
		logger.Printf("Database connection failed, running without database: %v", err)
		db = nil
		return nil
	}

	logger.Println("Database connected successfully")
	return nil
}

// Check if database is available
func hasDatabase() bool {
	return db != nil
}
