package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewAppConfiguresResourceApp(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	if app == nil {
		t.Fatal("newApp() returned nil app")
	}
	if app.CLI == nil {
		t.Fatal("newApp() returned nil CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("newApp() returned nil stale checker")
	}
	if app.StaleChecker.SourceContextPath != ".." {
		t.Fatalf("SourceContextPath = %q, want %q", app.StaleChecker.SourceContextPath, "..")
	}
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
	if len(app.StaleChecker.FreshnessInputs) != 3 {
		t.Fatalf("FreshnessInputs len = %d, want 3", len(app.StaleChecker.FreshnessInputs))
	}
	if got, want := app.StaleChecker.FreshnessInputs[0], "cli/**"; got != want {
		t.Fatalf("FreshnessInputs[0] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[1], "resource.json"; got != want {
		t.Fatalf("FreshnessInputs[1] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[2], "../../packages/cli-core"; got != want {
		t.Fatalf("FreshnessInputs[2] = %q, want %q", got, want)
	}
}

func TestPolicyCommandRegistered(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	output := captureStdout(t, func() {
		if err := app.CLI.Run([]string{"help"}); err != nil {
			t.Fatalf("help: %v", err)
		}
	})
	if !strings.Contains(output, "policy") {
		t.Fatalf("help output missing policy command:\n%s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}
