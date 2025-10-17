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
	
	fmt.Println("🔍 Verifying database tables...")
	
	// Check which tables exist
	tables := []string{
		"apis",
		"endpoints", 
		"pricing_tiers",
		"notes",
		"api_credentials",
		"research_requests",
		"api_usage_logs",
		"api_versions",
		"cost_calculations",
		"integration_snippets",
	}
	
	for _, table := range tables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)
		`, table).Scan(&exists)
		
		if err != nil {
			fmt.Printf("  ❌ Error checking %s: %v\n", table, err)
		} else if exists {
			// Get column count
			var colCount int
			db.QueryRow(`
				SELECT COUNT(*) FROM information_schema.columns 
				WHERE table_schema = 'public' AND table_name = $1
			`, table).Scan(&colCount)
			fmt.Printf("  ✅ %s (columns: %d)\n", table, colCount)
			
			// For notes table, check columns
			if table == "notes" {
				rows, _ := db.Query(`
					SELECT column_name FROM information_schema.columns 
					WHERE table_schema = 'public' AND table_name = 'notes'
					ORDER BY ordinal_position
				`)
				defer rows.Close()
				fmt.Printf("      Columns: ")
				for rows.Next() {
					var col string
					rows.Scan(&col)
					fmt.Printf("%s ", col)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("  ❌ %s (missing)\n", table)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}