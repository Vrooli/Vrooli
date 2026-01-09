package main

import "testing"

func TestAPIPathFromBaseURL(t *testing.T) {
	if got := apiPathFromBaseURL("http://localhost:1234", "health"); got != "/api/v1/health" {
		t.Fatalf("expected /api/v1/health, got %q", got)
	}
	if got := apiPathFromBaseURL("http://localhost:1234/api/v1", "/health"); got != "/health" {
		t.Fatalf("expected /health, got %q", got)
	}
	if got := apiPathFromBaseURL("http://localhost:1234/api/v1/", "health"); got != "/health" {
		t.Fatalf("expected /health, got %q", got)
	}
	if got := apiPathFromBaseURL(" http://localhost:1234 ", " "); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

