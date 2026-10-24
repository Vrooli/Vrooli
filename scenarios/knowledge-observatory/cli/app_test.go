package main

import "testing"

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	if app == nil {
		t.Fatal("expected app, got nil")
	}
	if app.core == nil {
		t.Fatal("expected core to be initialized")
	}
}

func TestRegisterCommands(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	groups := app.commandGroups()
	if len(groups) != 3 {
		t.Fatalf("expected 3 command groups, got %d", len(groups))
	}
	if groups[0].Title != "Health" {
		t.Fatalf("expected Health group, got %q", groups[0].Title)
	}
	if groups[1].Title != "Knowledge" {
		t.Fatalf("expected Knowledge group, got %q", groups[1].Title)
	}
	if groups[2].Title != "Documentation" {
		t.Fatalf("expected Documentation group, got %q", groups[2].Title)
	}
}

func TestAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	if got := app.core.APIPath(""); got != "" {
		t.Fatalf("expected empty path for empty input, got %q", got)
	}
	if got := app.core.APIPath("health"); got != "/api/v1/health" {
		t.Fatalf("expected /api/v1/health, got %q", got)
	}
}
