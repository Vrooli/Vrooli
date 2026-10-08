package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunRejectsUnknownFlag(t *testing.T) {
	if err := run([]string{"--unknown"}); err == nil {
		t.Fatal("run accepted an unknown flag")
	}
}

func TestMainReportsGenerationErrors(t *testing.T) {
	oldExit := exitProcess
	defer func() { exitProcess = oldExit }()
	called := 0
	exitProcess = func(code int) { called = code }
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"gen-endpoints", "--manifest", filepath.Join(t.TempDir(), "missing.json")}
	main()
	if called != 1 {
		t.Fatalf("exit code = %d, want 1", called)
	}
}
