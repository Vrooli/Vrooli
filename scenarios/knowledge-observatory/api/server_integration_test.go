package main

import (
	"os"
	"testing"
)

// [REQ:KO-SS-001,KO-SS-002,KO-SS-003] Test server creation error handling
func TestNewServerIntegration(t *testing.T) {
	t.Run("handles missing database config gracefully", func(t *testing.T) {
		oldPort := os.Getenv("API_PORT")
		oldDB := os.Getenv("DATABASE_URL")
		oldUser := os.Getenv("POSTGRES_USER")
		oldHost := os.Getenv("POSTGRES_HOST")
		oldPortVar := os.Getenv("POSTGRES_PORT")
		oldPassword := os.Getenv("POSTGRES_PASSWORD")
		oldDBName := os.Getenv("POSTGRES_DB")
		oldPostgresURL := os.Getenv("POSTGRES_URL")
		defer func() {
			if oldPort != "" {
				_ = os.Setenv("API_PORT", oldPort)
			} else {
				_ = os.Unsetenv("API_PORT")
			}
			if oldDB != "" {
				_ = os.Setenv("DATABASE_URL", oldDB)
			}
			if oldUser != "" {
				_ = os.Setenv("POSTGRES_USER", oldUser)
			}
			if oldHost != "" {
				_ = os.Setenv("POSTGRES_HOST", oldHost)
			}
			if oldPortVar != "" {
				_ = os.Setenv("POSTGRES_PORT", oldPortVar)
			}
			if oldPassword != "" {
				_ = os.Setenv("POSTGRES_PASSWORD", oldPassword)
			}
			if oldDBName != "" {
				_ = os.Setenv("POSTGRES_DB", oldDBName)
			}
			if oldPostgresURL != "" {
				_ = os.Setenv("POSTGRES_URL", oldPostgresURL)
			}
		}()

		_ = os.Setenv("API_PORT", "8080")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("POSTGRES_URL")
		_ = os.Unsetenv("POSTGRES_USER")
		_ = os.Unsetenv("POSTGRES_HOST")
		_ = os.Unsetenv("POSTGRES_PORT")
		_ = os.Unsetenv("POSTGRES_PASSWORD")
		_ = os.Unsetenv("POSTGRES_DB")

		_, err := NewServer()
		if err == nil {
			t.Error("NewServer() should return error when database config is missing")
		}
	})
}
