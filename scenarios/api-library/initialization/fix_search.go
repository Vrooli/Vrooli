package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	
	_ "github.com/lib/pq"
)

func main() {
	// Database connection details
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5433")
	dbUser := getEnv("POSTGRES_USER", "vrooli")
	dbPass := getEnv("POSTGRES_PASSWORD", "vrooli")
	dbName := getEnv("POSTGRES_DB", "api_library")
	
	// Connect to database
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)
	
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	fmt.Println("🔧 Fixing search vectors...")
	
	// Update search vectors for all existing rows
	query := `
		UPDATE apis
		SET search_vector = 
			setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
			setweight(to_tsvector('english', coalesce(provider, '')), 'B') ||
			setweight(to_tsvector('english', coalesce(description, '')), 'C') ||
			setweight(to_tsvector('english', coalesce(array_to_string(tags, ' '), '')), 'D')
	`
	
	result, err := db.Exec(query)
	if err != nil {
		log.Fatal("Failed to update search vectors:", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Updated search vectors for %d APIs\n", rowsAffected)
	
	// Verify by testing a search
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM apis 
		WHERE search_vector @@ plainto_tsquery('english', 'payment')
	`).Scan(&count)
	
	if err != nil {
		log.Fatal("Failed to test search:", err)
	}
	
	fmt.Printf("📊 Test search for 'payment' found %d results\n", count)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}