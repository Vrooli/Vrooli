// Package main contains CLI tests for reference-react-vite
// [REQ:MOD-P0-005] CLI as API wrapper - tests verify CLI initialization and structure
package main

import (
	"testing"
)

func TestAppCreation(t *testing.T) {
	t.Run("NewApp_creates_valid_app", func(t *testing.T) {
		app, err := NewApp()
		if err != nil {
			t.Fatalf("NewApp() failed: %v", err)
		}
		if app == nil {
			t.Fatal("expected app to be non-nil")
		}
		if app.core == nil {
			t.Fatal("expected app.core to be non-nil")
		}
	})
}

func TestAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty_path_returns_empty",
			input:    "",
			expected: "",
		},
		{
			name:     "path_with_leading_slash",
			input:    "/health",
			expected: "/api/v1/health",
		},
		{
			name:     "path_without_leading_slash",
			input:    "health",
			expected: "/api/v1/health",
		},
		{
			name:     "whitespace_only_returns_empty",
			input:    "   ",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := app.apiPath(tc.input)
			if result != tc.expected {
				t.Errorf("apiPath(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestHealthResponseParsing(t *testing.T) {
	t.Run("healthResponse_struct_exists", func(t *testing.T) {
		// Test that healthResponse struct can be instantiated
		resp := healthResponse{
			Status:    "healthy",
			Service:   "reference-react-vite",
			Version:   "0.1.0",
			Readiness: true,
			Timestamp: "2026-03-11T12:00:00Z",
			Deps: map[string]string{
				"postgres": "connected",
			},
		}

		if resp.Status != "healthy" {
			t.Errorf("expected Status 'healthy', got '%s'", resp.Status)
		}
		if !resp.Readiness {
			t.Error("expected Readiness to be true")
		}
		if resp.Deps["postgres"] != "connected" {
			t.Errorf("expected postgres dep 'connected', got '%s'", resp.Deps["postgres"])
		}
	})
}

func TestAppConstants(t *testing.T) {
	t.Run("constants_are_defined", func(t *testing.T) {
		if appName != "reference-react-vite" {
			t.Errorf("expected appName 'reference-react-vite', got '%s'", appName)
		}
		if appVersion == "" {
			t.Error("expected appVersion to be non-empty")
		}
	})
}
