package main

import (
	"reflect"
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
	// cli-core supplies the resource source inputs plus its own local package
	// path, which must participate in stale detection because it owns the
	// lifecycle command implementation.
	wantFreshnessInputs := []string{"cli/**", "resource.json", "../../packages/cli-core"}
	if !reflect.DeepEqual(app.StaleChecker.FreshnessInputs, wantFreshnessInputs) {
		t.Fatalf("FreshnessInputs = %q, want %q", app.StaleChecker.FreshnessInputs, wantFreshnessInputs)
	}
}
