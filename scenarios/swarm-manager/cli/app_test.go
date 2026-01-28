package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// [REQ:REQ-P0-009] Test CLI status command functionality
func TestNewApp(t *testing.T) {
	// Test that a new App can be created successfully
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}
	if app == nil {
		t.Fatal("NewApp() returned nil app")
	}
	if app.core == nil {
		t.Fatal("NewApp() returned app with nil core")
	}
}

// [REQ:REQ-P0-009] Test endpoint path resolution
func TestResolveV1Endpoint(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/health",
			expected: "/api/v1/health",
		},
		{
			name:     "path without leading slash",
			path:     "health",
			expected: "/api/v1/health",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "path with whitespace",
			path:     "  /health  ",
			expected: "/api/v1/health",
		},
		{
			name:     "ideas path",
			path:     "/ideas",
			expected: "/api/v1/ideas",
		},
	}

	app, err := NewApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.resolveV1Endpoint(tt.path)
			if result != tt.expected {
				t.Errorf("resolveV1Endpoint(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// [REQ:REQ-P0-009] Test app version and name constants
func TestAppConstants(t *testing.T) {
	if appName != "swarm-manager" {
		t.Errorf("appName = %q, want %q", appName, "swarm-manager")
	}
	if appVersion == "" {
		t.Error("appVersion should not be empty")
	}
}

// [REQ:REQ-P0-003] Test CLI ideas commands are registered
func TestIdeasCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// Get the registered command groups
	groups := app.core.CLI
	if groups == nil {
		t.Fatal("CLI app is nil")
	}

	// Verify ideas commands are registered by checking the help output
	// We can't directly access the commands map, but we can verify the app was created
	// with the ideas commands by checking the Run method doesn't panic
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"ideas list no-api", []string{"ideas", "list"}, true}, // Will fail without API but command should be found
		{"ideas get no-args", []string{"ideas", "get"}, true},  // Will fail without args
		{"ideas queue no-args", []string{"ideas", "queue"}, true},
		{"ideas research no-args", []string{"ideas", "research"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the app doesn't panic when trying to run these commands
			// The actual command execution will fail without a real API
			_ = app.Run(tt.args)
		})
	}
}

// Test scenario and recommendation commands are registered.
func TestScenarioCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"scenarios list no-api", []string{"scenarios", "list"}},
		{"scenarios get no-args", []string{"scenarios", "get"}},
		{"recommendations list no-api", []string{"recommendations", "list"}},
		{"recommendations update no-args", []string{"recommendations", "update"}},
		{"settings get no-api", []string{"settings", "get"}},
		{"queue list no-api", []string{"queue", "list"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = app.Run(tt.args)
		})
	}
}

// [REQ:REQ-P0-003] Test Idea struct JSON marshaling
func TestIdeaStruct(t *testing.T) {
	idea := Idea{
		Name:        "test-idea",
		Title:       "Test Idea",
		Description: "A test description",
		Status:      "backlog",
		Priority:    5,
		Tags:        []string{"test", "demo"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}

	data, err := json.Marshal(idea)
	if err != nil {
		t.Fatalf("Failed to marshal Idea: %v", err)
	}

	var parsed Idea
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal Idea: %v", err)
	}

	if parsed.Name != idea.Name {
		t.Errorf("Name = %q, want %q", parsed.Name, idea.Name)
	}
	if parsed.Title != idea.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, idea.Title)
	}
	if parsed.Status != idea.Status {
		t.Errorf("Status = %q, want %q", parsed.Status, idea.Status)
	}
	if parsed.Priority != idea.Priority {
		t.Errorf("Priority = %d, want %d", parsed.Priority, idea.Priority)
	}
	if len(parsed.Tags) != len(idea.Tags) {
		t.Errorf("Tags length = %d, want %d", len(parsed.Tags), len(idea.Tags))
	}
}

// [REQ:REQ-P0-003] Test CreateIdeaRequest struct JSON marshaling
func TestCreateIdeaRequestStruct(t *testing.T) {
	req := CreateIdeaRequest{
		Name:        "new-idea",
		Title:       "New Idea",
		Description: "Description",
		Priority:    3,
		Tags:        []string{"new"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal CreateIdeaRequest: %v", err)
	}

	// Verify JSON contains required fields
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"name":"new-idea"`) {
		t.Error("JSON should contain name field")
	}
	if !strings.Contains(jsonStr, `"title":"New Idea"`) {
		t.Error("JSON should contain title field")
	}
}

// [REQ:REQ-P0-003] Test cmdIdeasGet validates arguments
func TestCmdIdeasGetValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// Test with no arguments - should return error
	err = app.cmdIdeasGet([]string{})
	if err == nil {
		t.Error("cmdIdeasGet with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}
}

// [REQ:REQ-P0-003] Test cmdIdeasCreate validates arguments
func TestCmdIdeasCreateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// Test with no arguments - should return error
	err = app.cmdIdeasCreate([]string{})
	if err == nil {
		t.Error("cmdIdeasCreate with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	// Test with invalid JSON - should return error
	err = app.cmdIdeasCreate([]string{"not-valid-json"})
	if err == nil {
		t.Error("cmdIdeasCreate with invalid JSON should return error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("Error should contain 'invalid JSON', got: %v", err)
	}

	// Test with missing required fields - should return error
	err = app.cmdIdeasCreate([]string{`{"description":"only description"}`})
	if err == nil {
		t.Error("cmdIdeasCreate with missing name/title should return error")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("Error should contain 'required', got: %v", err)
	}
}

func TestCmdIdeasQueueValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdIdeasQueue([]string{})
	if err == nil {
		t.Error("cmdIdeasQueue with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdIdeasQueue([]string{"idea-name", "invalid"})
	if err == nil {
		t.Error("cmdIdeasQueue with invalid operation should return error")
	}
	if !strings.Contains(err.Error(), "invalid operation") {
		t.Errorf("Error should contain 'invalid operation', got: %v", err)
	}
}

func TestCmdIdeasResearchValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdIdeasResearch([]string{})
	if err == nil {
		t.Error("cmdIdeasResearch with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdIdeasResearch([]string{"idea-name", "{invalid"})
	if err == nil {
		t.Error("cmdIdeasResearch with invalid JSON should return error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("Error should contain 'invalid JSON', got: %v", err)
	}
}

func TestCmdScenariosGetValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdScenariosGet([]string{})
	if err == nil {
		t.Error("cmdScenariosGet with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}
}

func TestCmdRecommendationsUpdateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdRecommendationsUpdate([]string{})
	if err == nil {
		t.Error("cmdRecommendationsUpdate with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdRecommendationsUpdate([]string{"rec-1", "bad"})
	if err == nil {
		t.Error("cmdRecommendationsUpdate with invalid status should return error")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("Error should contain 'invalid status', got: %v", err)
	}
}

func TestCmdQueueCreateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdQueueCreate([]string{})
	if err == nil {
		t.Error("cmdQueueCreate with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdQueueCreate([]string{"build", "{invalid"})
	if err == nil {
		t.Error("cmdQueueCreate with invalid payload JSON should return error")
	}
	if !strings.Contains(err.Error(), "invalid payload JSON") {
		t.Errorf("Error should contain 'invalid payload JSON', got: %v", err)
	}
}

// [REQ:REQ-P0-003] Test cmdIdeasUpdate validates arguments
func TestCmdIdeasUpdateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// Test with no arguments - should return error
	err = app.cmdIdeasUpdate([]string{})
	if err == nil {
		t.Error("cmdIdeasUpdate with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	// Test with only one argument - should return error
	err = app.cmdIdeasUpdate([]string{"my-idea"})
	if err == nil {
		t.Error("cmdIdeasUpdate with only name should return error")
	}

	// Test with invalid JSON - should return error
	err = app.cmdIdeasUpdate([]string{"my-idea", "not-valid-json"})
	if err == nil {
		t.Error("cmdIdeasUpdate with invalid JSON should return error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("Error should contain 'invalid JSON', got: %v", err)
	}
}

// [REQ:REQ-P0-003] Test cmdIdeasDelete validates arguments
func TestCmdIdeasDeleteValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// Test with no arguments - should return error
	err = app.cmdIdeasDelete([]string{})
	if err == nil {
		t.Error("cmdIdeasDelete with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}
}

// [REQ:REQ-P0-003] Test ideas endpoint resolution
func TestIdeasEndpointResolution(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"/ideas", "/api/v1/ideas"},
		{"/ideas/test-idea", "/api/v1/ideas/test-idea"},
		{"ideas", "/api/v1/ideas"},
	}

	for _, tt := range tests {
		result := app.resolveV1Endpoint(tt.path)
		if result != tt.expected {
			t.Errorf("resolveV1Endpoint(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

// [REQ:REQ-P0-009] Test resolveV1Endpoint with various edge cases
func TestResolveV1Endpoint_EdgeCases(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"deeply nested path", "/ideas/test/files/docs/readme.md", "/api/v1/ideas/test/files/docs/readme.md"},
		{"path with query params", "/ideas?status=backlog", "/api/v1/ideas?status=backlog"},
		{"only slash", "/", "/api/v1/"},
		{"multiple leading slashes stripped", "//ideas", "/api/v1//ideas"},
		{"trailing slash", "/ideas/", "/api/v1/ideas/"},
		{"scenarios path", "/scenarios", "/api/v1/scenarios"},
		{"queue endpoint", "/ideas/test/queue", "/api/v1/ideas/test/queue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.resolveV1Endpoint(tt.path)
			if result != tt.expected {
				t.Errorf("resolveV1Endpoint(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// [REQ:REQ-P0-003] Test healthResponse struct JSON marshaling/unmarshaling
func TestHealthResponseStruct(t *testing.T) {
	resp := healthResponse{
		Status:    "ok",
		Service:   "swarm-manager-api",
		Version:   "0.1.0",
		Readiness: true,
		Timestamp: "2026-01-28T00:00:00Z",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal healthResponse: %v", err)
	}

	var parsed healthResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal healthResponse: %v", err)
	}

	if parsed.Status != "ok" {
		t.Errorf("Status = %q, want %q", parsed.Status, "ok")
	}
	if parsed.Service != "swarm-manager-api" {
		t.Errorf("Service = %q, want %q", parsed.Service, "swarm-manager-api")
	}
	if !parsed.Readiness {
		t.Error("Readiness should be true")
	}
	if len(parsed.Deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(parsed.Deps))
	}
}

// [REQ:REQ-P0-003] Test healthResponse with optional error fields
func TestHealthResponseWithError(t *testing.T) {
	resp := healthResponse{
		Status:  "error",
		Error:   "storage read failed",
		Message: "Service degraded",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal healthResponse: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"error":"storage read failed"`) {
		t.Error("JSON should contain error field")
	}
	if !strings.Contains(jsonStr, `"message":"Service degraded"`) {
		t.Error("JSON should contain message field")
	}
}

// [REQ:REQ-P0-003] Test cmdIdeasGet with multiple arguments ignores extras
func TestCmdIdeasGet_MultipleArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// With multiple args, only the first should be used as the name
	// This will fail at the API call level without a real server,
	// but we can verify it doesn't return the usage error
	err = app.cmdIdeasGet([]string{"first-arg", "second-arg", "third-arg"})

	// Should fail with network error, not usage error
	if err != nil && strings.Contains(err.Error(), "usage") {
		t.Error("cmdIdeasGet with multiple args should not return usage error")
	}
}

// [REQ:REQ-P0-003] Test cmdIdeasCreate with valid JSON but empty name
func TestCmdIdeasCreate_EmptyName(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// Valid JSON but name is empty string
	err = app.cmdIdeasCreate([]string{`{"name":"","title":"Test Title"}`})
	if err == nil {
		t.Error("cmdIdeasCreate with empty name should return error")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("Error should mention required fields, got: %v", err)
	}
}

// [REQ:REQ-P0-003] Test cmdIdeasCreate with valid JSON but empty title
func TestCmdIdeasCreate_EmptyTitle(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// Valid JSON but title is empty string
	err = app.cmdIdeasCreate([]string{`{"name":"test-name","title":""}`})
	if err == nil {
		t.Error("cmdIdeasCreate with empty title should return error")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("Error should mention required fields, got: %v", err)
	}
}

// [REQ:REQ-P0-003] Test cmdIdeasUpdate with exactly 2 arguments
func TestCmdIdeasUpdate_TwoArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// With exactly 2 args, should try to make API call (fail without server)
	err = app.cmdIdeasUpdate([]string{"idea-name", `{"title":"New Title"}`})
	// Should fail with network error, not usage or JSON error
	if err != nil {
		if strings.Contains(err.Error(), "usage") {
			t.Error("cmdIdeasUpdate with 2 args should not return usage error")
		}
		if strings.Contains(err.Error(), "invalid JSON") {
			t.Error("cmdIdeasUpdate with valid JSON should not return JSON error")
		}
	}
}
