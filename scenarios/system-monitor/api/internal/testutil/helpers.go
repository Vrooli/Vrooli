package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// WriteExecutableFile creates an executable file for test scenarios.
func WriteExecutableFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable file %q: %v", path, err)
	}
	return path
}

// DecodeJSONBody decodes a JSON byte slice into T and fails the test on error.
func DecodeJSONBody[T any](t *testing.T, body []byte) T {
	t.Helper()

	var parsed T
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decode json body: %v", err)
	}
	return parsed
}

// AssertStatusCode checks an HTTP status code and fails with context on mismatch.
func AssertStatusCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected status code: got %d want %d", got, want)
	}
}
