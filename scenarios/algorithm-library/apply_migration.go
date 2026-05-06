package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
	repocontract "github.com/vrooli/repo-contract-go"
)

func main() {
	// Use same env vars as the API
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5433"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "vrooli"
	}
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		log.Fatal("POSTGRES_PASSWORD environment variable is required")
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "vrooli"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Read migration file
	contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		log.Fatal("Failed to resolve repo contract:", err)
	}
	initDir, err := contract.ScenarioFile(repoRoot, "algorithm-library", "initialization")
	if err != nil {
		log.Fatal("Failed to resolve initialization path:", err)
	}
	migration, err := os.ReadFile(filepath.Join(initDir, "postgres", "migration_003_problem_mapping.sql"))
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	// Execute migration
	_, err = db.Exec(string(migration))
	if err != nil {
		log.Fatal("Failed to execute migration:", err)
	}

	fmt.Println("✅ Migration applied successfully!")
}
