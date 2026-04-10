package toolhandlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"scenario-to-desktop-api/toolregistry"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK TOOL PROVIDER
// =============================================================================

type mockToolProvider struct {
	name       string
	tools      []*toolspb.ToolDefinition
	categories []*toolspb.ToolCategory
}

func (m *mockToolProvider) Name() string                                         { return m.name }
func (m *mockToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition    { return m.tools }
func (m *mockToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory { return m.categories }

func newTestRegistry(tools ...*toolspb.ToolDefinition) *toolregistry.Registry {
	reg := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:    "test-scenario",
		ScenarioVersion: "1.0.0",
	})
	reg.RegisterProvider(&mockToolProvider{
		name:  "test-provider",
		tools: tools,
		categories: []*toolspb.ToolCategory{
			{Id: "test-cat", Name: "Test Category"},
		},
	})
	return reg
}

func newTestHandler(tools ...*toolspb.ToolDefinition) *ToolsHandler {
	return NewToolsHandler(newTestRegistry(tools...))
}

// =============================================================================
// extractPathParam
// =============================================================================

func TestExtractPathParam(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		prefix string
		want   string
	}{
		{"extracts name", "/api/v1/tools/my-tool", "/api/v1/tools/", "my-tool"},
		{"strips trailing slash", "/api/v1/tools/my-tool/", "/api/v1/tools/", "my-tool"},
		{"empty when path equals prefix", "/api/v1/tools/", "/api/v1/tools/", ""},
		{"empty when path shorter", "/api/v1/tools", "/api/v1/tools/", ""},
		{"handles nested names", "/api/v1/tools/ns.tool-name", "/api/v1/tools/", "ns.tool-name"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractPathParam(tc.path, tc.prefix)
			assert.Equal(t, tc.want, got)
		})
	}
}

// =============================================================================
// setCORSHeaders
// =============================================================================

func TestSetCORSHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	setCORSHeaders(w)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
}

// =============================================================================
// GET /api/v1/tools
// =============================================================================

func TestGetTools(t *testing.T) {
	sampleTool := &toolspb.ToolDefinition{
		Name:        "run-pipeline",
		Description: "Runs the build pipeline",
		Category:    "test-cat",
	}

	t.Run("returns manifest as JSON", func(t *testing.T) {
		h := newTestHandler(sampleTool)
		req := httptest.NewRequest("GET", "/api/v1/tools", nil)
		rec := httptest.NewRecorder()

		h.GetTools(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Equal(t, "public, max-age=60", rec.Header().Get("Cache-Control"))

		// Verify JSON is parseable and contains our tool
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "1.0", body["protocolVersion"])

		tools, ok := body["tools"].([]interface{})
		require.True(t, ok)
		assert.Len(t, tools, 1)
	})

	t.Run("sets CORS headers", func(t *testing.T) {
		h := newTestHandler(sampleTool)
		req := httptest.NewRequest("GET", "/api/v1/tools", nil)
		rec := httptest.NewRecorder()

		h.GetTools(rec, req)

		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		h := newTestHandler()
		req := httptest.NewRequest("OPTIONS", "/api/v1/tools", nil)
		rec := httptest.NewRecorder()

		h.GetTools(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("empty registry returns empty tools", func(t *testing.T) {
		h := NewToolsHandler(toolregistry.NewRegistry(toolregistry.RegistryConfig{
			ScenarioName: "empty",
		}))
		req := httptest.NewRequest("GET", "/api/v1/tools", nil)
		rec := httptest.NewRecorder()

		h.GetTools(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		// Should still have protocolVersion and scenario info
		assert.Equal(t, "1.0", body["protocolVersion"])
	})
}

// =============================================================================
// GET /api/v1/tools/{name}
// =============================================================================

func TestGetTool(t *testing.T) {
	sampleTool := &toolspb.ToolDefinition{
		Name:        "check-status",
		Description: "Checks pipeline status",
		Category:    "test-cat",
	}

	t.Run("returns specific tool", func(t *testing.T) {
		h := newTestHandler(sampleTool)
		req := httptest.NewRequest("GET", "/api/v1/tools/check-status", nil)
		req = mux.SetURLVars(req, map[string]string{"name": "check-status"})
		rec := httptest.NewRecorder()

		h.GetTool(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "check-status", body["name"])
		assert.Equal(t, "Checks pipeline status", body["description"])
	})

	t.Run("404 for unknown tool", func(t *testing.T) {
		h := newTestHandler(sampleTool)
		req := httptest.NewRequest("GET", "/api/v1/tools/nonexistent", nil)
		rec := httptest.NewRecorder()

		h.GetTool(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 for missing tool name", func(t *testing.T) {
		h := newTestHandler()
		req := httptest.NewRequest("GET", "/api/v1/tools/", nil)
		rec := httptest.NewRecorder()

		h.GetTool(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		h := newTestHandler()
		req := httptest.NewRequest("OPTIONS", "/api/v1/tools/anything", nil)
		rec := httptest.NewRecorder()

		h.GetTool(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

// =============================================================================
// ROUTE REGISTRATION
// =============================================================================

func TestRegisterRoutes(t *testing.T) {
	sampleTool := &toolspb.ToolDefinition{
		Name:        "test-tool",
		Description: "A test tool",
	}
	h := newTestHandler(sampleTool)
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	t.Run("GET /api/v1/tools is routed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tools", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GET /api/v1/tools/{name} is routed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tools/test-tool", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
