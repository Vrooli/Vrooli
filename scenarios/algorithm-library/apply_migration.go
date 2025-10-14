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
	migration, err := ioutil.ReadFile("/home/matthalloran8/Vrooli/scenarios/algorithm-library/initialization/postgres/migration_003_problem_mapping.sql")
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