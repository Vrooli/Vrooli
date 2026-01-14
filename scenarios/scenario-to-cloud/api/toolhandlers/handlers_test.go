package toolhandlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"scenario-to-cloud/toolregistry"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// mockToolProvider is a test double for toolregistry.ToolProvider.
type mockToolProvider struct {
	tools      []*toolspb.ToolDefinition
	categories []*toolspb.ToolCategory
}

func (m *mockToolProvider) Name() string {
	return "mock-provider"
}

func (m *mockToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return m.tools
}

func (m *mockToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return m.categories
}

func setupTestToolsHandler() (*ToolsHandler, *toolregistry.Registry) {
	registry := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "scenario-to-cloud",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Test scenario for unit tests",
	})

	provider := &mockToolProvider{
		tools: []*toolspb.ToolDefinition{
			{
				Name:        "create_deployment",
				Description: "Create a new deployment",
				Category:    "deployment_lifecycle",
				Parameters: &toolspb.ToolParameters{
					Type:       "object",
					Properties: map[string]*toolspb.ParameterSchema{},
				},
				Metadata: &toolspb.ToolMetadata{
					EnabledByDefault: true,
					TimeoutSeconds:   30,
				},
			},
			{
				Name:        "check_deployment_status",
				Description: "Check deployment status",
				Category:    "deployment_status",
				Parameters: &toolspb.ToolParameters{
					Type: "object",
					Properties: map[string]*toolspb.ParameterSchema{
						"deployment": {Type: "string", Description: "Deployment identifier"},
					},
					Required: []string{"deployment"},
				},
				Metadata: &toolspb.ToolMetadata{
					EnabledByDefault: true,
					TimeoutSeconds:   10,
				},
			},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "deployment_lifecycle", Name: "Deployment Lifecycle", Description: "Create and manage deployments"},
			{Id: "deployment_status", Name: "Deployment Status", Description: "Check deployment status"},
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
	var manifest toolspb.ToolManifest
	if err := protojson.Unmarshal(rec.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify manifest structure
	if manifest.ProtocolVersion != toolregistry.ToolProtocolVersion {
		t.Errorf("expected protocol version %s, got %s", toolregistry.ToolProtocolVersion, manifest.ProtocolVersion)
	}
	if manifest.Scenario.Name != "scenario-to-cloud" {
		t.Errorf("expected scenario name 'scenario-to-cloud', got %s", manifest.Scenario.Name)
	}
	if len(manifest.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(manifest.Tools))
	}
	if len(manifest.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(manifest.Categories))
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

	// Verify CORS headers are set
	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %s", allowOrigin)
	}
}

func TestToolsHandler_GetTool_Found(t *testing.T) {
	handler, _ := setupTestToolsHandler()

	req := httptest.NewRequest("GET", "/api/v1/tools/create_deployment", nil)
	rec := httptest.NewRecorder()

	handler.GetTool(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var tool toolspb.ToolDefinition
	if err := protojson.Unmarshal(rec.Body.Bytes(), &tool); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if tool.Name != "create_deployment" {
		t.Errorf("expected tool name 'create_deployment', got %s", tool.Name)
	}
	if tool.Description != "Create a new deployment" {
		t.Errorf("expected description 'Create a new deployment', got %s", tool.Description)
	}
}

func TestToolsHandler_GetTool_NotFound(t *testing.T) {
	handler, _ := setupTestToolsHandler()

	req := httptest.NewRequest("GET", "/api/v1/tools/nonexistent_tool", nil)
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

	req := httptest.NewRequest("OPTIONS", "/api/v1/tools/create_deployment", nil)
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
		{"/api/v1/tools/create_deployment", "/api/v1/tools/", "create_deployment"},
		{"/api/v1/tools/", "/api/v1/tools/", ""},
		{"/api/v1/tools", "/api/v1/tools/", ""},
		{"/api/v1/tools/a/b/c", "/api/v1/tools/", "a/b/c"},
		{"/api/v1/tools/my_tool/", "/api/v1/tools/", "my_tool"},
	}

	for _, tt := range tests {
		result := extractPathParam(tt.path, tt.prefix)
		if result != tt.expected {
			t.Errorf("extractPathParam(%q, %q) = %q, want %q", tt.path, tt.prefix, result, tt.expected)
		}
	}
}

// TestToolsHandler_WithRealProviders tests the handler with actual tool providers.
func TestToolsHandler_WithRealProviders(t *testing.T) {
	registry := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "scenario-to-cloud",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Deploys scenarios to VPS targets",
	})

	// Register all real providers
	registry.RegisterProvider(toolregistry.NewDeploymentToolProvider())
	registry.RegisterProvider(toolregistry.NewInspectionToolProvider())
	registry.RegisterProvider(toolregistry.NewValidationToolProvider())

	handler := NewToolsHandler(registry)

	req := httptest.NewRequest("GET", "/api/v1/tools", nil)
	rec := httptest.NewRecorder()

	handler.GetTools(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var manifest toolspb.ToolManifest
	if err := protojson.Unmarshal(rec.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should have all 11 tools
	expectedTools := 11
	if len(manifest.Tools) != expectedTools {
		t.Errorf("expected %d tools, got %d", expectedTools, len(manifest.Tools))
	}

	// Should have 4 categories
	expectedCategories := 4
	if len(manifest.Categories) != expectedCategories {
		t.Errorf("expected %d categories, got %d", expectedCategories, len(manifest.Categories))
	}
}
