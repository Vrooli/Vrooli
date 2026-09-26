package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// WriteJSONFile creates a JSON file at the specified path, creating parent
// directories if they don't exist.
func WriteJSONFile(tb testing.TB, path string, data any) {
	tb.Helper()
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		tb.Fatalf("create parent directory %s: %v", parentDir, err)
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		tb.Fatalf("marshal JSON for %s: %v", path, err)
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		tb.Fatalf("write file %s: %v", path, err)
	}
}

// WriteFile creates a file at the specified path with the given content,
// creating parent directories if they don't exist.
func WriteFile(tb testing.TB, path string, content string) {
	tb.Helper()
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		tb.Fatalf("create parent directory %s: %v", parentDir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatalf("write file %s: %v", path, err)
	}
}

// MakeDir creates a directory (and any missing parents) at the specified path.
func MakeDir(tb testing.TB, path string) {
	tb.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		tb.Fatalf("create directory %s: %v", path, err)
	}
}

// ReadJSONFile reads and decodes a JSON file into the provided type.
func ReadJSONFile[T any](tb testing.TB, path string) T {
	tb.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read file %s: %v", path, err)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		tb.Fatalf("decode JSON from %s: %v", path, err)
	}
	return result
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
