// Package handlers provides HTTP handlers for tool discovery endpoints.
//
// This file implements the Tool Discovery Protocol handlers.
// These endpoints enable external consumers (like agent-inbox) to discover
// what tools this scenario provides.
package handlers

import (
	"net/http"
	"strings"

	"github.com/vrooli/browser-automation-studio/internal/toolregistry"

	"google.golang.org/protobuf/encoding/protojson"
)

// ToolsHandler handles tool discovery endpoints.
type ToolsHandler struct {
	registry *toolregistry.Registry
}

// NewToolsHandler creates a new ToolsHandler.
func NewToolsHandler(registry *toolregistry.Registry) *ToolsHandler {
	return &ToolsHandler{registry: registry}
}

// protojson marshaler configured for compatibility with agent-inbox
var protoMarshaler = protojson.MarshalOptions{
	UseProtoNames:   false, // Use camelCase field names for JSON
	EmitUnpopulated: false, // Don't emit zero values
}

// GetTools handles GET /api/v1/tools
// Returns the complete tool manifest for this scenario.
func (h *ToolsHandler) GetTools(w http.ResponseWriter, r *http.Request) {
	manifest := h.registry.GetManifest(r.Context())

	// Use protojson for proper proto JSON serialization
	data, err := protoMarshaler.Marshal(manifest)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60") // Cache for 1 minute
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// GetTool handles GET /api/v1/tools/{name}
// Returns a specific tool definition by name.
func (h *ToolsHandler) GetTool(w http.ResponseWriter, r *http.Request) {
	// Extract tool name from URL path
	// Chi router passes the full path, so we need to extract the tool name
	name := extractToolName(r.URL.Path)
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "tool name is required")
		return
	}

	tool := h.registry.GetTool(r.Context(), name)
	if tool == nil {
		writeJSONError(w, http.StatusNotFound, "tool not found: "+name)
		return
	}

	// Use protojson for proper proto JSON serialization
	data, err := protoMarshaler.Marshal(tool)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// extractToolName extracts the tool name from paths like /api/v1/tools/{name}
func extractToolName(path string) string {
	// Handle both /api/v1/tools/{name} and /tools/{name} patterns
	prefixes := []string{"/api/v1/tools/", "/tools/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			name := strings.TrimPrefix(path, prefix)
			// Don't return "execute" as a tool name - that's a different endpoint
			if name != "" && name != "execute" {
				return name
			}
		}
	}
	return ""
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + http.StatusText(status) + `","message":"` + message + `"}`))
}
