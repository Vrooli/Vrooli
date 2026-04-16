package main

import (
	"testing"

	resourceapp "resource-gemini/cli/internal/app"
)

func TestNewConfiguresResourceApp(t *testing.T) {
	app, err := resourceapp.New(buildFingerprint, buildTimestamp, buildSourceRoot)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app == nil {
		t.Fatal("New() returned nil app")
	}
	if app.CLI == nil {
		t.Fatal("New() returned nil CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("New() returned nil stale checker")
	}
	if app.StaleChecker.SourceContextPath != ".." {
		t.Fatalf("SourceContextPath = %q, want %q", app.StaleChecker.SourceContextPath, "..")
	}
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
}
