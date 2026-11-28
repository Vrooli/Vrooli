package main

import "testing"

func TestNewAppConfiguresResourceApp(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	if app == nil || app.CLI == nil {
		t.Fatal("newApp() returned nil app or CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("newApp() returned nil stale checker")
	}
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
}

func TestCapacityStepsLadder(t *testing.T) {
	steps := capacitySteps()
	if len(steps) != 2 {
		t.Fatalf("expected 2 capacity rungs, got %d", len(steps))
	}
	if steps[0].Label != "model" || steps[1].Label != "regex-only" {
		t.Errorf("unexpected rung labels: %q, %q", steps[0].Label, steps[1].Label)
	}
	for _, s := range steps {
		if s.Action != "restart" {
			t.Errorf("rung %q action = %q, want restart", s.Label, s.Action)
		}
	}
}
