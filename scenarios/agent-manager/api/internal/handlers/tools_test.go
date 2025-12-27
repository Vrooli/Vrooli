package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/toolregistry"
)

// mockToolProvider is a test double for toolregistry.ToolProvider.
type mockToolProvider struct {
	tools      []domain.ToolDefinition
	categories []domain.ToolCategory
}

func (m *mockToolProvider) Name() string {
	return "mock-provider"
}

func (m *mockToolProvider) Tools(_ context.Context) []domain.ToolDefinition {
	return m.tools
}

func (m *mockToolProvider) Categories(_ context.Context) []domain.ToolCategory {
	return m.categories
}

func setupTestToolsHandler() (*ToolsHandler, *toolregistry.Registry) {
	registry := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "test-scenario",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Test scenario for unit tests",
	})

	provider := &mockToolProvider{
		tools: []domain.ToolDefinition{
			{
				Name:        "test_tool",
				Description: "A test tool",
				Category:    "testing",
				Parameters: domain.ToolParameters{
					Type:       "object",
					Properties: map[string]domain.ParameterSchema{},
				},
				Metadata: domain.ToolMetadata{
					EnabledByDefault: true,
					TimeoutSeconds:   30,
				},
			},
			{
				Name:        "another_tool",
				Description: "Another test tool",
				Category:    "testing",
				Parameters: domain.ToolParameters{
					Type: "object",
					Properties: map[string]domain.ParameterSchema{
						"input": {Type: "string", Description: "Input value"},
					},
					Required: []string{"input"},
				},
				Metadata: domain.ToolMetadata{
					EnabledByDefault: true,
					TimeoutSeconds:   60,
				},
			},
		},
		categories: []domain.ToolCategory{
			{ID: "testing", Name: "Testing", Description: "Tools for testing"},
		},
	}

	registry.RegisterProvider(provider)
	handler := NewToolsHandler(registry)

	return handler, registry
}

func TestToolsHandler_GetTools(t *testing.T) {
	handler, _ := setupTestToolsHandler()

	req := httptest.NewRequest("GET", "/api/v1/tools", nil)
	rec := httptest.NewRecorder()

	handler.GetTools(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}

	// Verify cache control
	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=60" {
		t.Errorf("expected Cache-Control 'public, max-age=60', got %s", cacheControl)
	}

	// Parse response
	var manifest domain.ToolManifest
	if err := json.Unmarshal(rec.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify manifest structure
	if manifest.ProtocolVersion != domain.ProtocolVersion {
		t.Errorf("expected protocol version %s, got %s", domain.ProtocolVersion, manifest.ProtocolVersion)
	}
	if manifest.Scenario.Name != "test-scenario" {
		t.Errorf("expected scenario name 'test-scenario', got %s", manifest.Scenario.Name)
	}
	if len(manifest.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(manifest.Tools))
	}
	if len(manifest.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(manifest.Categories))
	}
}

func TestToolsHandler_GetTools_OPTIONS(t *testing.T) {
	handler, _ := setupTestToolsHandler()

	req := httptest.NewRequest("OPTIONS", "/api/v1/tools", nil)
	rec := httptest.NewRecorder()

	handler.GetTools(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for OPTIONS, got %d", rec.Code)
	}
}

func TestToolsHandler_GetTool_Found(t *testing.T) {
	handler, _ := setupTestToolsHandler()

	req := httptest.NewRequest("GET", "/api/v1/tools/test_tool", nil)
	rec := httptest.NewRecorder()

	handler.GetTool(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var tool domain.ToolDefinition
	if err := json.Unmarshal(rec.Body.Bytes(), &tool); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if tool.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %s", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %s", tool.Description)
	}
}

func TestToolsHandler_GetTool_NotFound(t *testing.T) {
	handler, _ := setupTestToolsHandler()

	req := httptest.NewRequest("GET", "/api/v1/tools/nonexistent", nil)
	rec := httptest.NewRecorder()

	handler.GetTool(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	var errResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if errResp["error"] != "Not Found" {
		t.Errorf("expected error 'Not Found', got %s", errResp["error"])
	}
}

func TestToolsHandler_GetTool_EmptyName(t *testing.T) {
	handler, _ := setupTestToolsHandler()

	req := httptest.NewRequest("GET", "/api/v1/tools/", nil)
	rec := httptest.NewRecorder()

	handler.GetTool(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestToolsHandler_GetTool_OPTIONS(t *testing.T) {
	handler, _ := setupTestToolsHandler()

	req := httptest.NewRequest("OPTIONS", "/api/v1/tools/test_tool", nil)
	rec := httptest.NewRecorder()

	handler.GetTool(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for OPTIONS, got %d", rec.Code)
	}
}

func TestExtractPathParam(t *testing.T) {
	tests := []struct {
		path     string
		prefix   string
		expected string
	}{
		{"/api/v1/tools/my_tool", "/api/v1/tools/", "my_tool"},
		{"/api/v1/tools/", "/api/v1/tools/", ""},
		{"/api/v1/tools", "/api/v1/tools/", ""},
		{"/api/v1/tools/a/b/c", "/api/v1/tools/", "a/b/c"},
	}

	for _, tt := range tests {
		result := extractPathParam(tt.path, tt.prefix)
		if result != tt.expected {
			t.Errorf("extractPathParam(%q, %q) = %q, want %q", tt.path, tt.prefix, result, tt.expected)
		}
	}
}
