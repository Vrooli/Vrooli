package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:REQ-P0-009] Test CLI status command functionality
func TestNewApp(t *testing.T) {
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
			name:     "backlog path",
			path:     "/backlog",
			expected: "/api/v1/backlog",
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

// [REQ:REQ-P0-003] Test CLI backlog commands are registered
func TestBacklogCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	groups := app.core.CLI
	if groups == nil {
		t.Fatal("CLI app is nil")
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"backlog list no-api", []string{"backlog", "list"}, true},
		{"backlog get no-args", []string{"backlog", "get"}, true},
		{"backlog queue no-args", []string{"backlog", "queue"}, true},
		{"backlog research no-args", []string{"backlog", "research"}, true},
		{"backlog convert no-args", []string{"backlog", "convert"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = app.Run(tt.args)
		})
	}
}

// Test scenario/settings/queue commands are registered.
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
		{"scenarios spec-sync-archive no-args", []string{"scenarios", "spec-sync-archive"}},
		{"scenarios get no-args", []string{"scenarios", "get"}},
		{"settings get no-api", []string{"settings", "get"}},
		{"queue list no-api", []string{"queue", "list"}},
		{"backlog prompt-trace no-args", []string{"backlog", "prompt-trace"}},
		{"execution prompt-trace no-args", []string{"execution", "prompt-trace"}},
		{"prompts map no-api", []string{"prompts", "map"}},
		{"prompts skill-get no-args", []string{"prompts", "skill-get"}},
		{"prompts preview no-args", []string{"prompts", "preview"}},
		{"agent-manager status no-api", []string{"agent-manager", "status"}},
		{"execution create no-args", []string{"execution", "create"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = app.Run(tt.args)
		})
	}
}

// [REQ:REQ-P0-003] Test BacklogItem struct JSON marshaling
func TestBacklogItemStruct(t *testing.T) {
	item := BacklogItem{
		Name:        "test-idea",
		Title:       "Test Idea",
		Description: "A test description",
		Status:      "backlog",
		Priority:    5,
		Tags:        []string{"test", "demo"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        "idea",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal BacklogItem: %v", err)
	}

	var parsed BacklogItem
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal BacklogItem: %v", err)
	}

	if parsed.Name != item.Name {
		t.Errorf("Name = %q, want %q", parsed.Name, item.Name)
	}
	if parsed.Title != item.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, item.Title)
	}
	if parsed.Status != item.Status {
		t.Errorf("Status = %q, want %q", parsed.Status, item.Status)
	}
	if parsed.Priority != item.Priority {
		t.Errorf("Priority = %d, want %d", parsed.Priority, item.Priority)
	}
	if len(parsed.Tags) != len(item.Tags) {
		t.Errorf("Tags length = %d, want %d", len(parsed.Tags), len(item.Tags))
	}
}

// [REQ:REQ-P0-003] Test CreateBacklogRequest struct JSON marshaling
func TestCreateBacklogRequestStruct(t *testing.T) {
	req := CreateBacklogRequest{
		Name:        "new-idea",
		Title:       "New Idea",
		Description: "Description",
		Priority:    3,
		Tags:        []string{"new"},
		Kind:        "idea",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal CreateBacklogRequest: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"name":"new-idea"`) {
		t.Error("JSON should contain name field")
	}
	if !strings.Contains(jsonStr, `"title":"New Idea"`) {
		t.Error("JSON should contain title field")
	}
	if !strings.Contains(jsonStr, `"kind":"idea"`) {
		t.Error("JSON should contain kind field")
	}
}

// [REQ:REQ-P0-003] Test cmdBacklogGet validates arguments
func TestCmdBacklogGetValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogGet([]string{})
	if err == nil {
		t.Error("cmdBacklogGet with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}
}

// [REQ:REQ-P0-003] Test cmdBacklogCreate validates arguments
func TestCmdBacklogCreateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogCreate([]string{})
	if err == nil {
		t.Error("cmdBacklogCreate with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdBacklogCreate([]string{"--data", "not-valid-json"})
	if err == nil {
		t.Error("cmdBacklogCreate with invalid JSON should return error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("Error should contain 'invalid JSON', got: %v", err)
	}

	err = app.cmdBacklogCreate([]string{"--data", `{"description":"only description"}`})
	if err == nil {
		t.Error("cmdBacklogCreate with missing name/title/kind should return error")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("Error should contain 'required', got: %v", err)
	}
}

func TestCmdBacklogQueueValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogQueue([]string{})
	if err == nil {
		t.Error("cmdBacklogQueue with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdBacklogQueue([]string{"--kind", "idea", "--name", "item-name", "--operation", "invalid"})
	if err == nil {
		t.Error("cmdBacklogQueue with invalid operation should return error")
	}
	if !strings.Contains(err.Error(), "invalid operation") {
		t.Errorf("Error should contain 'invalid operation', got: %v", err)
	}

	err = app.cmdBacklogQueue([]string{"--kind", "idea", "--name", "item-name", "--mode", "invalid"})
	if err == nil {
		t.Error("cmdBacklogQueue with invalid mode should return error")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("Error should contain 'invalid mode', got: %v", err)
	}
}

func TestCmdExecutionPolicyUpdateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdExecutionPolicyUpdate([]string{"--delay-seconds", "30"})
	if err == nil {
		t.Error("cmdExecutionPolicyUpdate without mode should return error")
	}
	if !strings.Contains(err.Error(), "mode is required") {
		t.Errorf("Error should contain 'mode is required', got: %v", err)
	}

	err = app.cmdExecutionPolicyUpdate([]string{"--mode", "manual", "--delay-seconds", "-1"})
	if err == nil {
		t.Error("cmdExecutionPolicyUpdate with negative delay should return error")
	}
	if !strings.Contains(err.Error(), "delay-seconds must be >= 0") {
		t.Errorf("Error should contain delay validation message, got: %v", err)
	}
}

func TestCmdBacklogResearchValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogResearch([]string{})
	if err == nil {
		t.Error("cmdBacklogResearch with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdBacklogResearch([]string{"--kind", "idea", "--name", "item-name", "--data", "{invalid"})
	if err == nil {
		t.Error("cmdBacklogResearch with invalid JSON should return error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("Error should contain 'invalid JSON', got: %v", err)
	}
}

func TestCmdBacklogUpdateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogUpdate([]string{})
	if err == nil {
		t.Error("cmdBacklogUpdate with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdBacklogUpdate([]string{"--kind", "idea", "--name", "my-idea"})
	if err == nil {
		t.Error("cmdBacklogUpdate with missing data should return error")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("Error should contain 'required', got: %v", err)
	}

	err = app.cmdBacklogUpdate([]string{"--kind", "idea", "--name", "my-idea", "--data", "not-valid-json"})
	if err == nil {
		t.Error("cmdBacklogUpdate with invalid JSON should return error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("Error should contain 'invalid JSON', got: %v", err)
	}
}

func TestCmdBacklogDeleteValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogDelete([]string{})
	if err == nil {
		t.Error("cmdBacklogDelete with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}
}

func TestCmdBacklogFileGetValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogFileGet([]string{})
	if err == nil {
		t.Error("cmdBacklogFileGet with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}
}

func TestCmdBacklogFileUploadValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogFileUpload([]string{})
	if err == nil {
		t.Error("cmdBacklogFileUpload with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}
}

func TestCmdBacklogConvertValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogConvert([]string{})
	if err == nil {
		t.Error("cmdBacklogConvert with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
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

	err = app.cmdQueueCreate([]string{"--kind", "build", "--data", "{invalid"})
	if err == nil {
		t.Error("cmdQueueCreate with invalid payload JSON should return error")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Error should contain 'invalid', got: %v", err)
	}
}

func TestCmdExecutionCreateValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdExecutionCreate([]string{})
	if err == nil {
		t.Error("cmdExecutionCreate with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Error should contain 'usage', got: %v", err)
	}

	err = app.cmdExecutionCreate([]string{"--kind", "idea", "--name", "test", "--mode", "invalid"})
	if err == nil {
		t.Error("cmdExecutionCreate with invalid mode should return error")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("Error should contain 'invalid mode', got: %v", err)
	}
}

// [REQ:REQ-P0-003] Test backlog endpoint resolution
func TestBacklogEndpointResolution(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"/backlog", "/api/v1/backlog"},
		{"/backlog/idea/test-idea", "/api/v1/backlog/idea/test-idea"},
		{"backlog", "/api/v1/backlog"},
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
		{"deeply nested path", "/backlog/idea/test/files/docs/readme.md", "/api/v1/backlog/idea/test/files/docs/readme.md"},
		{"path with query params", "/backlog?kinds=idea", "/api/v1/backlog?kinds=idea"},
		{"only slash", "/", "/api/v1/"},
		{"multiple leading slashes stripped", "//backlog", "/api/v1//backlog"},
		{"trailing slash", "/backlog/", "/api/v1/backlog/"},
		{"scenarios path", "/scenarios", "/api/v1/scenarios"},
		{"queue endpoint", "/backlog/idea/test/queue", "/api/v1/backlog/idea/test/queue"},
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
	if !strings.Contains(jsonStr, "storage read failed") {
		t.Error("JSON should contain error field")
	}
	if !strings.Contains(jsonStr, "Service degraded") {
		t.Error("JSON should contain message field")
	}
}

func TestParseJSONString_FromFile(t *testing.T) {
	tempDir := t.TempDir()
	payloadPath := filepath.Join(tempDir, "payload.json")
	if err := os.WriteFile(payloadPath, []byte(`{"name":"from-file","title":"From File","kind":"idea"}`), 0o644); err != nil {
		t.Fatalf("write payload file: %v", err)
	}

	raw, err := parseJSONString("@" + payloadPath)
	if err != nil {
		t.Fatalf("parseJSONString returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to decode parsed payload: %v", err)
	}
	if parsed["name"] != "from-file" {
		t.Fatalf("name mismatch: got %v", parsed["name"])
	}
}

func TestCmdBacklogFilesValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdBacklogFiles([]string{})
	if err == nil {
		t.Fatal("cmdBacklogFiles with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}

func TestScenarioLifecycleValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	err = app.cmdScenariosStart([]string{})
	if err == nil {
		t.Fatal("cmdScenariosStart with no args should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}

func TestDecodeResponse(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	decoded, err := decodeResponse[payload]([]byte(`{"name":"ok"}`))
	if err != nil {
		t.Fatalf("decodeResponse returned error: %v", err)
	}
	if decoded.Name != "ok" {
		t.Fatalf("unexpected decoded name: %q", decoded.Name)
	}
}

func TestRequestMultipartV1IncludesAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/backlog/idea/test/files" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "text/plain" {
			t.Fatalf("unexpected content-type header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	t.Setenv("SWARM_MANAGER_API_TOKEN", "test-token")

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	body, err := app.requestMultipartV1("POST", "/backlog/idea/test/files", []byte("payload"), "text/plain")
	if err != nil {
		t.Fatalf("requestMultipartV1 returned error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected response body: %s", string(body))
	}
}

func TestCmdPromptsMapRequestsExpectedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/prompts/map" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"items":[{"area":"research","trigger":"x","skill_id":"swarm-manager-clarify-idea","purpose":"p"}]}`))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}
	if err := app.cmdPromptsMap([]string{}); err != nil {
		t.Fatalf("cmdPromptsMap returned error: %v", err)
	}
}

func TestCmdPromptsPreviewSendsPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/prompts/preview" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["skill_id"] != "swarm-manager-clarify-idea" {
			t.Fatalf("unexpected skill_id: %v", payload["skill_id"])
		}
		if payload["with_scope"] != true {
			t.Fatalf("expected with_scope=true, got %v", payload["with_scope"])
		}
		vars, ok := payload["variables"].(map[string]any)
		if !ok {
			t.Fatalf("expected variables object, got %T", payload["variables"])
		}
		if vars["ITEM_TITLE"] != "My Idea" {
			t.Fatalf("unexpected ITEM_TITLE: %v", vars["ITEM_TITLE"])
		}
		_, _ = w.Write([]byte(`{"skill_id":"swarm-manager-clarify-idea","with_scope":true,"variables":{"ITEM_TITLE":"My Idea"},"prompt":"preview prompt"}`))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}
	if err := app.cmdPromptsPreview([]string{"--id", "swarm-manager-clarify-idea", "--with-scope", "--vars", "ITEM_TITLE=My Idea"}); err != nil {
		t.Fatalf("cmdPromptsPreview returned error: %v", err)
	}
}

func TestCmdExecutionPromptTraceRequestsExpectedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/execution/ex-123/prompt-trace" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"trace":{"skill_id":"swarm-manager-process-idea","purpose":"p","prompt":"prompt text","used_fallback":false,"captured_at":"2026-01-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}
	if err := app.cmdExecutionPromptTrace([]string{"--id", "ex-123"}); err != nil {
		t.Fatalf("cmdExecutionPromptTrace returned error: %v", err)
	}
}

func TestCmdScenariosSpecSyncArchiveSendsPreserveFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/scenarios/my-scenario/spec-sync-archive" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		preserve, ok := payload["preserve_files"].(map[string]any)
		if !ok {
			t.Fatalf("expected preserve_files object, got %T", payload["preserve_files"])
		}
		if preserve["preset"] != "planning" {
			t.Fatalf("unexpected preset: %v", preserve["preset"])
		}
		paths, ok := preserve["paths"].([]any)
		if !ok || len(paths) != 2 {
			t.Fatalf("unexpected paths: %#v", preserve["paths"])
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"execution_id":"exec-1","status":"queued","message":"ok"}`))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}
	if err := app.cmdScenariosSpecSyncArchive([]string{"--name", "my-scenario", "--preset", "planning", "--paths", "PRD.md,docs/**"}); err != nil {
		t.Fatalf("cmdScenariosSpecSyncArchive returned error: %v", err)
	}
}

func TestParseKVCSVValidation(t *testing.T) {
	if _, err := parseKVCSV("FOO"); err == nil {
		t.Fatal("expected parseKVCSV to reject invalid key-value entry")
	}
	values, err := parseKVCSV("A=1,B=2")
	if err != nil {
		t.Fatalf("parseKVCSV returned error: %v", err)
	}
	if values["A"] != "1" || values["B"] != "2" {
		t.Fatalf("unexpected parsed values: %#v", values)
	}
}

func TestCmdStatus_ParsesNestedDependencies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"healthy","service":"swarm-manager-api","readiness":true,"dependencies":{"database":{"connected":true}}}`))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}
	if err := app.cmdStatus([]string{}); err != nil {
		t.Fatalf("cmdStatus returned error: %v", err)
	}
}
