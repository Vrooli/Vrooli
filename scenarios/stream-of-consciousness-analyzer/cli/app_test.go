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

// [REQ:P0-001] Test App version is set
func TestAppConstants(t *testing.T) {
	if appName == "" {
		t.Error("appName must be set")
	}
	if appVersion == "" {
		t.Error("appVersion must be set")
	}
	if appName != "stream-of-consciousness-analyzer" {
		t.Errorf("expected appName=stream-of-consciousness-analyzer, got %s", appName)
	}
}

// [REQ:P0-001] Test apiPath handles various input formats
func TestApiPathEdgeCases(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"with leading slash", "/test", "/api/v1/test"},
		{"without leading slash", "test", "/api/v1/test"},
		{"nested path", "/test/nested/deep", "/api/v1/test/nested/deep"},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := app.apiPath(tc.input)
			if got != tc.expected {
				t.Errorf("apiPath(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// [REQ:P0-003] Test info create requires both scheme and content
func TestInfoCreateRequiresBothFlags(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"no args at all", nil},
		{"only scheme", []string{"--scheme", "s1"}},
		{"only content", []string{"--content", "hello"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := app.cmdInfoCreate(tc.args)
			if err == nil {
				t.Error("expected error for incomplete arguments")
			}
		})
	}
}

// [REQ:P0-003] Test info update requires scheme and at least one field
func TestInfoUpdateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"id only", []string{"some-id"}},
		{"id and scheme but no fields", []string{"some-id", "--scheme", "s1"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := app.cmdInfoUpdate(tc.args)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

// [REQ:P0-003] Test info delete requires scheme and id
func TestInfoDeleteValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"id only no scheme", []string{"some-id"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := app.cmdInfoDelete(tc.args)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

// [REQ:P0-001] Test command group count matches expected
func TestCommandGroupCount(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	groups := app.registerCommands()
	if len(groups) != 7 {
		t.Errorf("expected 7 command groups, got %d", len(groups))
	}
}

// [REQ:P0-004] Test edge create requires source and target
func TestEdgeCreateFullValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"source only", []string{"source-id"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := app.cmdEdgeCreate(tc.args)
			if err == nil {
				t.Error("expected error for missing arguments")
			}
		})
	}
}
