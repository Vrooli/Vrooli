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
	
	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	fmt.Println("🔧 Initializing api-library database...")
	
	// Schema files to apply in order
	schemaFiles := []struct {
		file string
		desc string
	}{
		{"postgres/schema.sql", "main schema"},
		{"postgres/schema_update_v2.sql", "v2 updates"},
		{"postgres/schema_integration_snippets.sql", "integration snippets"},
		{"postgres/seed.sql", "initial API data"},
	}
	
	for _, schema := range schemaFiles {
		fmt.Printf("  → Applying %s...\n", schema.desc)
		
		sqlFile := schema.file
		content, err := ioutil.ReadFile(sqlFile)
		if err != nil {
			fmt.Printf("    ⚠️  Could not read %s: %v\n", schema.file, err)
			continue
		}
		
		_, err = db.Exec(string(content))
		if err != nil {
			// Don't fail on errors as tables might already exist
			fmt.Printf("    ⚠️  Some warnings while applying %s (continuing...)\n", schema.desc)
		} else {
			fmt.Printf("    ✅ %s applied successfully\n", schema.desc)
		}
	}
	
	// Get statistics
	var apiCount, snippetCount int
	db.QueryRow("SELECT COUNT(*) FROM apis").Scan(&apiCount)
	db.QueryRow("SELECT COUNT(*) FROM integration_snippets").Scan(&snippetCount)
	
	fmt.Println("\n✅ Database initialization complete!")
	fmt.Println("\n📊 Database statistics:")
	fmt.Printf("  • APIs loaded: %d\n", apiCount)
	fmt.Printf("  • Snippets loaded: %d\n", snippetCount)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}