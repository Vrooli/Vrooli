package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
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
	
	fmt.Println("🔧 Applying table fixes...")
	
	// Read and execute fix SQL
	content, err := ioutil.ReadFile("fix_notes_table.sql")
	if err != nil {
		log.Fatal("Failed to read fix SQL:", err)
	}
	
	_, err = db.Exec(string(content))
	if err != nil {
		log.Printf("Error applying fixes: %v", err)
	} else {
		fmt.Println("✅ Tables fixed successfully")
	}
	
	// Verify the fix
	var hasApiId bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'notes' 
			AND column_name = 'api_id'
		)
	`).Scan(&hasApiId)
	
	if hasApiId {
		fmt.Println("✅ notes table has api_id column")
	} else {
		fmt.Println("❌ notes table missing api_id column")
	}
	
	var researchExists bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'research_requests'
		)
	`).Scan(&researchExists)
	
	if researchExists {
		fmt.Println("✅ research_requests table exists")
	} else {
		fmt.Println("❌ research_requests table missing")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}