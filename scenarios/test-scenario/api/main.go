package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// Test scenario with hardcoded secrets (for testing vulnerability scanner)
const (
	// These are intentional hardcoded secrets for testing the scanner
	DatabasePassword = "scenario_hardcoded_password_456"
	APIKey          = "sk-scenario-test-abcdef1234567890123456789012345678901234567890"
	JWTSecret       = "scenario_jwt_secret_key_testing_67890"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start test-scenario

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Hardcoded database URL with credentials
	dbURL := "postgres://user:hardcoded123@localhost/testdb"
	
	log.Printf("Connecting to database: %s", dbURL)
	log.Printf("Using API key: %s", APIKey)
	
	http.ListenAndServe(":8080", nil)
}