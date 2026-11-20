package main

import "testing"

func TestNewAppConfiguresResourceLifecycle(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	if app == nil || app.CLI == nil {
		t.Fatal("newApp() did not return an initialized resource CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("newApp() returned no stale checker")
	}
	if app.StaleChecker.SourceContextPath != ".." {
		t.Fatalf("SourceContextPath = %q, want %q", app.StaleChecker.SourceContextPath, "..")
	}
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
}
