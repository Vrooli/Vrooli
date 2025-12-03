// Package config tests
// [REQ:FAIL-SAFE-001]
package config

import (
	"os"
	"testing"
)

func TestLoad_WithDatabaseURL(t *testing.T) {
	// Save and restore environment
	origDBURL := os.Getenv("DATABASE_URL")
	origAPIPort := os.Getenv("API_PORT")
	defer func() {
		os.Setenv("DATABASE_URL", origDBURL)
		os.Setenv("API_PORT", origAPIPort)
	}()

	// Set test values
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	os.Setenv("API_PORT", "8080")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}

	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://user:pass@localhost:5432/testdb")
	}
}

func TestLoad_WithPostgresEnvVars(t *testing.T) {
	// Save and restore environment
	envVars := []string{"DATABASE_URL", "API_PORT", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"}
	origVals := make(map[string]string)
	for _, v := range envVars {
		origVals[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range origVals {
			os.Setenv(k, v)
		}
	}()

	// Clear DATABASE_URL to force individual env var resolution
	os.Unsetenv("DATABASE_URL")
	os.Setenv("API_PORT", "9000")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_USER", "testuser")
	os.Setenv("POSTGRES_PASSWORD", "testpass")
	os.Setenv("POSTGRES_DB", "testdb")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9000")
	}

	// Verify URL is constructed correctly (contains all parts)
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL should not be empty")
	}

	// Check that URL contains expected parts
	expectedParts := []string{"postgres://", "testuser", "localhost:5432", "testdb"}
	for _, part := range expectedParts {
		if !containsSubstring(cfg.DatabaseURL, part) {
			t.Errorf("DatabaseURL %q should contain %q", cfg.DatabaseURL, part)
		}
	}
}

func TestLoad_MissingAPIPort(t *testing.T) {
	// Save and restore environment
	origAPIPort := os.Getenv("API_PORT")
	origDBURL := os.Getenv("DATABASE_URL")
	defer func() {
		os.Setenv("API_PORT", origAPIPort)
		os.Setenv("DATABASE_URL", origDBURL)
	}()

	os.Unsetenv("API_PORT")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when API_PORT is not set")
	}
}

func TestLoad_MissingDatabaseConfig(t *testing.T) {
	// Save and restore environment
	envVars := []string{"DATABASE_URL", "API_PORT", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"}
	origVals := make(map[string]string)
	for _, v := range envVars {
		origVals[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range origVals {
			os.Setenv(k, v)
		}
	}()

	// Clear all database-related env vars
	for _, v := range envVars {
		os.Unsetenv(v)
	}
	os.Setenv("API_PORT", "8080")

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when no database configuration is available")
	}
}

func TestRequireEnv(t *testing.T) {
	// Save and restore
	origVal := os.Getenv("TEST_VAR_UNIQUE_12345")
	defer os.Setenv("TEST_VAR_UNIQUE_12345", origVal)

	os.Setenv("TEST_VAR_UNIQUE_12345", "  test-value  ")
	result := requireEnv("TEST_VAR_UNIQUE_12345")
	if result != "test-value" {
		t.Errorf("requireEnv() = %q, want %q (should trim whitespace)", result, "test-value")
	}

	os.Unsetenv("TEST_VAR_UNIQUE_12345")
	result = requireEnv("TEST_VAR_UNIQUE_12345")
	if result != "" {
		t.Errorf("requireEnv() = %q, want empty string for unset var", result)
	}
}

func TestResolveDatabaseURL_DirectURL(t *testing.T) {
	// Save and restore
	origDBURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", origDBURL)

	os.Setenv("DATABASE_URL", "  postgres://user:pass@host/db  ")

	result, err := resolveDatabaseURL()
	if err != nil {
		t.Fatalf("resolveDatabaseURL() error = %v", err)
	}
	if result != "postgres://user:pass@host/db" {
		t.Errorf("resolveDatabaseURL() = %q, want %q (should trim whitespace)", result, "postgres://user:pass@host/db")
	}
}

func TestResolveDatabaseURL_PartialConfig(t *testing.T) {
	// Save and restore environment
	envVars := []string{"DATABASE_URL", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"}
	origVals := make(map[string]string)
	for _, v := range envVars {
		origVals[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range origVals {
			os.Setenv(k, v)
		}
	}()

	// Clear all, then set only some
	for _, v := range envVars {
		os.Unsetenv(v)
	}
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	// Missing: USER, PASSWORD, DB

	_, err := resolveDatabaseURL()
	if err == nil {
		t.Error("resolveDatabaseURL() should fail when partial config is provided")
	}
}

// containsSubstring is a helper for checking URL parts
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
