package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// Database initialization with exponential backoff
func initDB(logger *log.Logger) error {
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	
	// Check if all required env vars are set
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		// Database is optional for time-tools, can work without it
		logger.Println("Database configuration not found, running without database")
		return nil
	}
	
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	var err error
	maxRetries := 5
	baseDelay := time.Second
	
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				logger.Println("Database connected successfully")
				return nil
			}
		}
		
		delay := time.Duration(math.Pow(2, float64(i))) * baseDelay
		logger.Printf("Database connection failed (attempt %d/%d), retrying in %v: %v", 
			i+1, maxRetries, delay, err)
		time.Sleep(delay)
	}
	
	// Database connection failed, but we can still run without it
	logger.Printf("Database connection failed after %d attempts, running without database: %v", maxRetries, err)
	db = nil
	return nil
}

// Check if database is available
func hasDatabase() bool {
	return db != nil
}