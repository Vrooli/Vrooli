package main

import "testing"

func TestAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	if got := app.core.APIPath("health"); got != "/api/v1/health" {
		t.Fatalf("expected /api/v1/health, got %q", got)
	}
	if got := app.core.APIPath(" "); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
