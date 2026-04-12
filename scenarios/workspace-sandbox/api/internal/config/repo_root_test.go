package config

import (
	"os"
	"testing"
)

func TestResolveDefaultProjectRoot(t *testing.T) {
	origProjectRoot := os.Getenv("PROJECT_ROOT")
	t.Cleanup(func() {
		if origProjectRoot == "" {
			os.Unsetenv("PROJECT_ROOT")
		} else {
			os.Setenv("PROJECT_ROOT", origProjectRoot)
		}
	})

	t.Run("prefers explicit project root", func(t *testing.T) {
		os.Setenv("PROJECT_ROOT", "/tmp/custom")
		if got := ResolveDefaultProjectRoot(); got != "/tmp/custom" {
			t.Fatalf("ResolveDefaultProjectRoot() = %q, want %q", got, "/tmp/custom")
		}
	})
}
