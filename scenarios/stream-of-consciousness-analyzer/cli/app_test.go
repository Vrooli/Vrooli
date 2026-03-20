package main

import "testing"

// [REQ:P0-001] Test CLI app initialization
func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	if app == nil {
		t.Fatal("expected non-nil app")
	}
}

// [REQ:P0-001] Test API path construction
func TestApiPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	cases := []struct {
		input    string
		expected string
	}{
		{"/health", "/api/v1/health"},
		{"health", "/api/v1/health"},
		{"/schemes", "/api/v1/schemes"},
		{"/thoughts", "/api/v1/thoughts"},
		{"/providers", "/api/v1/providers"},
		{"", ""},
	}

	for _, tc := range cases {
		got := app.apiPath(tc.input)
		if got != tc.expected {
			t.Errorf("apiPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// [REQ:P0-001] [REQ:P0-002] Test CLI registers all expected command groups
func TestCommandRegistration(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	if app.core == nil {
		t.Fatal("expected non-nil core")
	}

	groups := app.registerCommands()
	expectedGroups := []string{"Health", "Schemes", "Thoughts", "Edges", "Information", "Suggestions", "Configuration"}
	if len(groups) != len(expectedGroups) {
		t.Errorf("expected %d command groups, got %d", len(expectedGroups), len(groups))
	}

	for i, eg := range expectedGroups {
		if i < len(groups) && groups[i].Title != eg {
			t.Errorf("group %d: expected title %q, got %q", i, eg, groups[i].Title)
		}
	}
}

// [REQ:P0-001] Test CLI scheme commands require arguments
func TestSchemeCommandsValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name string
		fn   func([]string) error
		args []string
	}{
		{"scheme get no args", app.cmdSchemeGet, nil},
		{"scheme update no args", app.cmdSchemeUpdate, nil},
		{"scheme delete no args", app.cmdSchemeDelete, nil},
		{"scheme export no args", app.cmdSchemeExport, nil},
		{"scheme update no name", app.cmdSchemeUpdate, []string{"some-id"}},
		{"scheme create no name", app.cmdSchemeCreate, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn(tc.args)
			if err == nil {
				t.Error("expected error for missing arguments")
			}
		})
	}
}

// [REQ:P0-004] Test CLI thought commands require arguments
func TestThoughtCommandsValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name string
		fn   func([]string) error
		args []string
	}{
		{"thought get no args", app.cmdThoughtGet, nil},
		{"thought create no title", app.cmdThoughtCreate, nil},
		{"thought update no args", app.cmdThoughtUpdate, nil},
		{"thought update no fields", app.cmdThoughtUpdate, []string{"some-id"}},
		{"thought delete no args", app.cmdThoughtDelete, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn(tc.args)
			if err == nil {
				t.Error("expected error for missing arguments")
			}
		})
	}
}

// [REQ:P0-004] Test CLI edge commands require arguments
func TestEdgeCommandsValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name string
		fn   func([]string) error
		args []string
	}{
		{"edge list no args", app.cmdEdgeList, nil},
		{"edge create no args", app.cmdEdgeCreate, nil},
		{"edge create no target", app.cmdEdgeCreate, []string{"source-id"}},
		{"edge delete no args", app.cmdEdgeDelete, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn(tc.args)
			if err == nil {
				t.Error("expected error for missing arguments")
			}
		})
	}
}

// [REQ:P0-003] Test CLI info commands require arguments
func TestInfoCommandsValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name string
		fn   func([]string) error
		args []string
	}{
		{"info list no args", app.cmdInfoList, nil},
		{"info create no scheme", app.cmdInfoCreate, nil},
		{"info update no args", app.cmdInfoUpdate, nil},
		{"info delete no args", app.cmdInfoDelete, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn(tc.args)
			if err == nil {
				t.Error("expected error for missing arguments")
			}
		})
	}
}

// [REQ:P1-001] Test CLI suggestion commands require arguments
func TestSuggestionCommandsValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = app.cmdSuggestionGenerate(nil)
	if err == nil {
		t.Error("expected error for missing scheme ID")
	}
}
