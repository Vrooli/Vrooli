package main

import (
	"os"
	"testing"
)

// [REQ:KO-SS-001,KO-SS-002,KO-SS-003] Test server creation error handling
func TestNewServerIntegration(t *testing.T) {
	t.Run("handles missing database config gracefully", func(t *testing.T) {
		oldDB := os.Getenv("DATABASE_URL")
		oldUser := os.Getenv("POSTGRES_USER")
		defer func() {
			if oldDB != "" {
				_ = os.Setenv("DATABASE_URL", oldDB)
			}
			if oldUser != "" {
				_ = os.Setenv("POSTGRES_USER", oldUser)
			}
		}()

		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("POSTGRES_USER")

		_, err := NewServer()
		if err == nil {
			t.Error("NewServer() should return error when database config is missing")
		}
	})
}
