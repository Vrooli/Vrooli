package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesAndChecksReference(t *testing.T) {
	output := filepath.Join(t.TempDir(), "reference.md")
	if err := run([]string{"--output", output}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--output", output, "--check"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--output", output, "--check"}); err == nil {
		t.Fatal("stale reference was accepted")
	}
}

func TestMainReportsReferenceErrors(t *testing.T) {
	oldExit := exitProcess
	defer func() { exitProcess = oldExit }()
	called := 0
	exitProcess = func(code int) { called = code }
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test-quality-reference", "--output", filepath.Join(t.TempDir(), "missing", "reference.md")}
	main()
	if called != 1 {
		t.Fatalf("exit code = %d, want 1", called)
	}
}
