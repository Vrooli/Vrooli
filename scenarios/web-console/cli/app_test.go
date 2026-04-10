package main

import (
	"testing"
)

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}
	if app == nil {
		t.Fatal("expected non-nil app")
	}
	if app.core == nil {
		t.Fatal("expected non-nil core")
	}
}

func TestApiPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"/health", "/api/v1/health"},
		{"health", "/api/v1/health"},
		{"", ""},
	}

	for _, tc := range tests {
		got := app.apiPath(tc.input)
		if got != tc.want {
			t.Errorf("apiPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
