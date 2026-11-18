package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMainGeneratesEndpointsFromFlags(t *testing.T) {
	output := filepath.Join(t.TempDir(), "endpoints.json")
	oldArgs, oldExit := os.Args, exitProcess
	defer func() {
		os.Args, exitProcess = oldArgs, oldExit
	}()
	var exitCode int
	exitProcess = func(code int) { exitCode = code }
	os.Args = []string{"gen-endpoints", "--output", output, "--manifest", filepath.Join("..", "..", "..", "cli", "manifest.json")}

	main()
	if exitCode != 0 {
		t.Fatalf("main() exit code = %d, want 0", exitCode)
	}
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated endpoints: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(contents, &document); err != nil {
		t.Fatalf("generated endpoints are not JSON: %v", err)
	}
	if len(document) == 0 {
		t.Fatal("generated endpoints document is empty")
	}

	exitCode = 0
	os.Args = []string{"gen-endpoints", "--output", filepath.Join(t.TempDir(), "failed.json"), "--manifest", filepath.Join(t.TempDir(), "missing-manifest.json")}
	main()
	if exitCode != 1 {
		t.Fatalf("main() exit code for invalid manifest = %d, want 1", exitCode)
	}
}

func TestRunReportsInvalidFlags(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"--unknown"}, &stderr); code == 0 {
		t.Fatalf("run() exit code = %d, want non-zero", code)
	}
	if stderr.Len() == 0 {
		t.Fatal("invalid flags produced no diagnostic")
	}
}
