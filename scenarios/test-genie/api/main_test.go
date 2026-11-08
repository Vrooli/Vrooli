package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("TG_API_EARLY_WORKER_HELPER") == "1" {
		// Exercise the production entry, with every API dependency initialized,
		// rather than only the seed package's smaller test executable.
		main()
		os.Exit(99)
	}
	os.Exit(m.Run())
}

func TestSeedWorkerModeRejectsBeforeAPIOrRepositoryInitialization(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "invalid.input.json")
	if err := os.WriteFile(input, []byte(`{"version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "--test-genie-seed-worker", input, "invalid-input-hash")
	command.WaitDelay = time.Second
	// This matches the native worker's deliberately minimal bootstrap env.
	// No repo-root, scenario, API port, storage or credential may be required.
	command.Env = []string{"PATH=/usr/bin:/bin", "TG_API_EARLY_WORKER_HELPER=1"}
	output, err := command.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 || !bytes.Contains(output, []byte("seed native worker refused or failed")) {
		t.Fatalf("production worker entry did not refuse before API initialization: %v", err)
	}
	if bytes.Contains(output, []byte("panic:")) || bytes.Contains(output, []byte("VROOLI_ROOT")) {
		t.Fatal("early worker attempted repository-dependent API initialization")
	}
}
