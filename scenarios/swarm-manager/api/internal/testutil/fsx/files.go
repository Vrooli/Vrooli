package fsx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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

func MakeDir(tb testing.TB, path string) {
	tb.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		tb.Fatalf("create directory %s: %v", path, err)
	}
}

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

func AssertFileExists(tb testing.TB, path string) {
	tb.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		tb.Errorf("expected file to exist: %s", path)
	}
}

func AssertFileNotExists(tb testing.TB, path string) {
	tb.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		tb.Errorf("expected file to not exist: %s", path)
	}
}
