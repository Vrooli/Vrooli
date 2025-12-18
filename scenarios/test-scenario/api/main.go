package main

import (
	"github.com/vrooli/api-core/preflight"
	"log"
	"net/http"
)

// Test scenario with hardcoded secrets (for testing vulnerability scanner)
const (
	// These are intentional hardcoded secrets for testing the scanner
	DatabasePassword = "scenario_hardcoded_password_456"
	APIKey          = "sk-scenario-test-abcdef1234567890123456789012345678901234567890"
	JWTSecret       = "scenario_jwt_secret_key_testing_67890"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "test-scenario",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Hardcoded database URL with credentials
	dbURL := "postgres://user:hardcoded123@localhost/testdb"
	
	log.Printf("Connecting to database: %s", dbURL)
	log.Printf("Using API key: %s", APIKey)
	
	http.ListenAndServe(":8080", nil)
}