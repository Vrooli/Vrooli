package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCompanionConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop.json")
	if err := os.WriteFile(path, []byte(`{"display_id":"macos-current-user-display"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateCompanionConfig(path); err != nil {
		t.Fatalf("valid companion config rejected: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"display_id":"`+strings.Repeat("x", 257)+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateCompanionConfig(path); err == nil {
		t.Fatal("oversized display id accepted")
	}
}
