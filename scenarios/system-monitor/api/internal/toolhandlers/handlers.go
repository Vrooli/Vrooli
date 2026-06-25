// Package toolhandlers provides HTTP handlers for the Tool Discovery Protocol.
package toolhandlers

import (
	"log/slog"
	"net/http"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/toolregistry"
)

// ToolsHandler handles HTTP requests for tool discovery.
type ToolsHandler struct {
	registry *toolregistry.Registry
	log      *slog.Logger
}

// NewToolsHandler creates a new tool discovery handler.
func NewToolsHandler(registry *toolregistry.Registry, log *slog.Logger) *ToolsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &ToolsHandler{
		registry: registry,
		log:      log,
	}
}

// GetManifest handles GET /api/v1/tools
// Returns the complete tool manifest with all available tools.
func (h *ToolsHandler) GetManifest(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("fetching tool manifest")

	manifest := h.registry.GetManifest(r.Context())
	h.writeJSON(w, manifest, http.StatusOK)
}

// GetTool handles GET /api/v1/tools/{name}
// Returns a specific tool definition by name.
func (h *ToolsHandler) GetTool(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	h.log.Debug("fetching tool", "name", name)

	tool := h.registry.GetTool(r.Context(), name)
	if tool == nil {
		h.writeError(w, "tool not found: "+name, http.StatusNotFound)
		return
	}

	h.writeJSON(w, tool, http.StatusOK)
}

// RegisterRoutes registers the tool discovery routes on the given router.
func (h *ToolsHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("GET /api/v1/tools", h.GetManifest)
	r.HandleFunc("OPTIONS /api/v1/tools", h.GetManifest)
	r.HandleFunc("GET /api/v1/tools/{name}", h.GetTool)
	r.HandleFunc("OPTIONS /api/v1/tools/{name}", h.GetTool)
}

// writeJSON writes a JSON response with the given status code through the
// shared httputil writer so the secure header floor is applied centrally.
func (h *ToolsHandler) writeJSON(w http.ResponseWriter, data interface{}, status int) {
	if err := httputil.JSONWithStatus(w, status, data); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

// writeError writes a JSON error response.
func (h *ToolsHandler) writeError(w http.ResponseWriter, message string, status int) {
	h.writeJSON(w, map[string]interface{}{
		"error":  message,
		"status": "error",
	}, status)
}
