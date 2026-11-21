package testutil

import (
	"os"
	"testing"
)

// MakeDir creates a directory (and any missing parents) at the specified path.
func MakeDir(tb testing.TB, path string) {
	tb.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		tb.Fatalf("create directory %s: %v", path, err)
	}
}

// AssertFileExists checks that a file exists at the specified path.
func AssertFileExists(tb testing.TB, path string) {
	tb.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		tb.Errorf("expected file to exist: %s", path)
	}
}

// AssertFileNotExists checks that a file does not exist at the specified path.
func AssertFileNotExists(tb testing.TB, path string) {
	tb.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		tb.Errorf("expected file to not exist: %s", path)
	}
}
