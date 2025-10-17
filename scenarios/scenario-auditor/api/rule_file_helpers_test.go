package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func sanitizeRuleForTest(t *testing.T, relativePath string) string {
	t.Helper()

	content, err := os.ReadFile(relativePath)
	if err != nil {
		t.Fatalf("failed to read rule %s: %v", relativePath, err)
	}

	cleaned := stripMetadataComment(content)

	tempRoot := t.TempDir()
	configDir := filepath.Join(tempRoot, "config")
	if err := os.MkdirAll(configDir, fs.ModePerm); err != nil {
		t.Fatalf("failed to create temp config dir: %v", err)
	}

	dest := filepath.Join(configDir, filepath.Base(relativePath))
	if err := os.WriteFile(dest, cleaned, 0o600); err != nil {
		t.Fatalf("failed to write temp rule copy: %v", err)
	}

	typesSrc := filepath.Join(filepath.Dir(filepath.Dir(relativePath)), "types.go")
	if data, err := os.ReadFile(typesSrc); err == nil {
		if err := os.WriteFile(filepath.Join(tempRoot, "types.go"), data, 0o600); err != nil {
			t.Fatalf("failed to copy types.go: %v", err)
		}
	}

	return dest
}

func stripMetadataComment(content []byte) []byte {
	start := bytes.Index(content, []byte("/*"))
	end := bytes.Index(content, []byte("*/"))
	if start >= 0 && end > start {
		trimmed := append([]byte{}, content[:start]...)
		trimmed = append(trimmed, content[end+2:]...)
		return trimmed
	}
	return content
}
