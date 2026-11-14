package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/proto/contracts"
)

func TestReadUntilInputAcceptsLargeFileAndRejectsOversize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goal.txt")
	want := strings.Repeat("x", contracts.AgentManagerMaxGoalCompletionBytes)
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readUntilInput("", path)
	if err != nil || got != want {
		t.Fatalf("max-size file got %d bytes, err=%v", len(got), err)
	}

	if err := os.WriteFile(path, []byte(want+"x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readUntilInput("", path); err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("oversized file error = %v", err)
	}
}

func TestReadUntilInputRejectsConflictingSourcesAndInvalidUTF8(t *testing.T) {
	if _, err := readUntilInput("inline", "goal.txt"); err == nil {
		t.Fatal("inline and file inputs should be mutually exclusive")
	}
	path := filepath.Join(t.TempDir(), "invalid.txt")
	if err := os.WriteFile(path, []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readUntilInput("", path); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}
