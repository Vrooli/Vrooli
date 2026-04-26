// Package handlers provides HTTP handlers for the agent-manager API.
//
// This file implements the Tool Discovery Protocol handlers.
// These endpoints enable external consumers to discover what tools
// this scenario provides.
package handlers

import (
	"net/http"

	"agent-manager/internal/toolregistry"

	"google.golang.org/protobuf/encoding/protojson"
)

// ToolsHandler handles tool discovery endpoints.
// Separated from the main Handler to maintain clear boundaries.
type ToolsHandler struct {
	registry *toolregistry.Registry
}

// NewToolsHandler creates a new ToolsHandler.
func NewToolsHandler(registry *toolregistry.Registry) *ToolsHandler {
	return &ToolsHandler{registry: registry}
}

// RegisterRoutes registers the tool discovery routes.
func (h *ToolsHandler) RegisterRoutes(r RouteRegistrar) {
	r.HandleFunc("/api/v1/tools", h.GetTools).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/tools/{name}", h.GetTool).Methods("GET", "OPTIONS")
}

// RouteRegistrar is an interface for registering routes.
// This enables testing without depending on gorilla/mux directly.
type RouteRegistrar interface {
	HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) RouteMethoder
}

// RouteMethoder allows chaining Methods() after HandleFunc.
type RouteMethoder interface {
	Methods(methods ...string) RouteMethoder
}

// protojson marshaler configured for compatibility with OpenAI format
var protoMarshaler = protojson.MarshalOptions{
	UseProtoNames:   false, // Use camelCase field names for JSON
	EmitUnpopulated: false, // Don't emit zero values
}

// GetTools handles GET /api/v1/tools
// Returns the complete tool manifest for this scenario.
func (h *ToolsHandler) GetTools(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

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
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Extract tool name from path
	// Note: Using path parsing instead of mux.Vars to reduce coupling
	name := extractPathParam(r.URL.Path, "/api/v1/tools/")
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

// extractPathParam extracts the parameter after a prefix from the URL path.
func extractPathParam(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	return path[len(prefix):]
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Use simple JSON for error responses
	_, _ = w.Write([]byte(`{"error":"` + http.StatusText(status) + `","message":"` + message + `"}`))
}
