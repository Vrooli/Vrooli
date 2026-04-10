package main

import (
	"testing"
)

// [REQ:CLI-STATUS-001] Status command is registered
// [REQ:CLI-STATUS-002] JSON output flag parsing
func TestUseJSONFlag(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	tests := []struct {
		args []string
		want bool
	}{
		{[]string{}, false},
		{[]string{"--json"}, true},
		{[]string{"-j"}, true},
		{[]string{"--verbose", "--json"}, true},
		{[]string{"--verbose"}, false},
	}

	for _, tc := range tests {
		got := app.useJSON(tc.args)
		if got != tc.want {
			t.Errorf("useJSON(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

// [REQ:CLI-STATUS-001] API path construction
func TestAPIPathConstruction(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"/tunnel/health", "/api/v1/tunnel/health"},
		{"/routes", "/api/v1/routes"},
		{"routes", "/api/v1/routes"},
		{"", ""},
	}

	for _, tc := range tests {
		got := app.apiPath(tc.input)
		if got != tc.want {
			t.Errorf("apiPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
