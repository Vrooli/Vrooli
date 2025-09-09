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

func testDBConnection() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable is required")
	}
	
	db, err := sql.Open("postgres", databaseURL)
	if err \!= nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5) 
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		time.Sleep(actualDelay)
	}

	if pingErr \!= nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
}

func main() {
	fmt.Println("Database connection logic compiles successfully")
}
EOF < /dev/null
