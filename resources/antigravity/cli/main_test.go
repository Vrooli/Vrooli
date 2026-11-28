package main

import (
	"testing"

	"github.com/vrooli/vrooli/resources/testkit"
)

func TestNewAppConfiguresResourceApp(t *testing.T) {
	h := testkit.Handlers(t)
	if h.Stdout == nil {
		t.Fatal("test harness must provide stdout")
	}
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
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
}
