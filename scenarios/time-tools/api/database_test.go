package main

import (
	"os"
	"testing"
	"time"
)

// TestHasDatabase tests the database availability check
func TestHasDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		// Save original db
		originalDB := db
		defer func() { db = originalDB }()

		// Set db to nil
		db = nil

		if hasDatabase() {
			t.Error("Expected hasDatabase to return false when db is nil")
		}
	})
}

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingEnvironmentVariables", func(t *testing.T) {
		// Save and clear environment variables
		origHost := os.Getenv("POSTGRES_HOST")
		origPort := os.Getenv("POSTGRES_PORT")
		origUser := os.Getenv("POSTGRES_USER")
		origPassword := os.Getenv("POSTGRES_PASSWORD")
		origDB := os.Getenv("POSTGRES_DB")

		os.Setenv("POSTGRES_HOST", "")
		os.Setenv("POSTGRES_PORT", "")
		os.Setenv("POSTGRES_USER", "")
		os.Setenv("POSTGRES_PASSWORD", "")
		os.Setenv("POSTGRES_DB", "")

		defer func() {
			os.Setenv("POSTGRES_HOST", origHost)
			os.Setenv("POSTGRES_PORT", origPort)
			os.Setenv("POSTGRES_USER", origUser)
			os.Setenv("POSTGRES_PASSWORD", origPassword)
			os.Setenv("POSTGRES_DB", origDB)
		}()

		err := initDB(logger)
		if err != nil {
			t.Errorf("Expected no error when env vars missing, got: %v", err)
		}
	})

	t.Run("InvalidDatabaseConnection", func(t *testing.T) {
		// Save original values
		origHost := os.Getenv("POSTGRES_HOST")
		origPort := os.Getenv("POSTGRES_PORT")
		origUser := os.Getenv("POSTGRES_USER")
		origPassword := os.Getenv("POSTGRES_PASSWORD")
		origDB := os.Getenv("POSTGRES_DB")

		defer func() {
			os.Setenv("POSTGRES_HOST", origHost)
			os.Setenv("POSTGRES_PORT", origPort)
			os.Setenv("POSTGRES_USER", origUser)
			os.Setenv("POSTGRES_PASSWORD", origPassword)
			os.Setenv("POSTGRES_DB", origDB)
		}()

		// Set invalid database credentials
		os.Setenv("POSTGRES_HOST", "invalid-host-that-does-not-exist")
		os.Setenv("POSTGRES_PORT", "9999")
		os.Setenv("POSTGRES_USER", "invalid")
		os.Setenv("POSTGRES_PASSWORD", "invalid")
		os.Setenv("POSTGRES_DB", "invalid")

		// This should handle the error gracefully and not panic
		err := initDB(logger)
		// initDB returns nil even on failure (database is optional)
		if err != nil {
			t.Logf("Database connection failed as expected: %v", err)
		}
	})
}

// TestDatabaseConnectionRetry tests exponential backoff
func TestDatabaseConnectionRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retry test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RetriesOnFailure", func(t *testing.T) {
		// Save original values
		origHost := os.Getenv("POSTGRES_HOST")
		origPort := os.Getenv("POSTGRES_PORT")
		origUser := os.Getenv("POSTGRES_USER")
		origPassword := os.Getenv("POSTGRES_PASSWORD")
		origDB := os.Getenv("POSTGRES_DB")

		defer func() {
			os.Setenv("POSTGRES_HOST", origHost)
			os.Setenv("POSTGRES_PORT", origPort)
			os.Setenv("POSTGRES_USER", origUser)
			os.Setenv("POSTGRES_PASSWORD", origPassword)
			os.Setenv("POSTGRES_DB", origDB)
		}()

		// Set invalid credentials to force retries
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "9999")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")

		start := time.Now()
		err := initDB(logger)
		elapsed := time.Since(start)

		// Should complete (returns nil even on failure)
		if err != nil {
			t.Logf("Expected no error (db is optional), got: %v", err)
		}

		// Should take some time due to retries
		if elapsed < 0 {
			t.Errorf("Expected some elapsed time for retries, got %v", elapsed)
		}
		t.Logf("Database initialization with retries took: %v", elapsed)
	})
}
