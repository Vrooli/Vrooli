// Package handlers provides HTTP handlers for app-monitor.
//
// This file provides handlers for the Tool Discovery Protocol endpoints.
package handlers

import (
	"net/http"

	"app-monitor-api/toolregistry"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
)

// ToolsHandler handles tool discovery endpoints.
type ToolsHandler struct {
	registry *toolregistry.Registry
}

// NewToolsHandler creates a new ToolsHandler with the given registry.
func NewToolsHandler(registry *toolregistry.Registry) *ToolsHandler {
	return &ToolsHandler{registry: registry}
}

// protoMarshalOptions configures protojson marshaling for tool responses.
var protoMarshalOptions = protojson.MarshalOptions{
	EmitUnpopulated: true,
	UseProtoNames:   false, // Use camelCase for JSON
}

// GetManifest handles GET /api/v1/tools - returns the complete tool manifest.
func (h *ToolsHandler) GetManifest(c *gin.Context) {
	manifest := h.registry.GetManifest(c.Request.Context())

	// Marshal to JSON using protojson for proper proto message handling
	jsonBytes, err := protoMarshalOptions.Marshal(manifest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to marshal manifest",
			"message": err.Error(),
		})
		return
	}

	// Set cache headers (1 minute cache for discovery)
	c.Header("Cache-Control", "public, max-age=60")
	c.Header("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write(jsonBytes)
}

// GetTool handles GET /api/v1/tools/:name - returns a specific tool definition.
func (h *ToolsHandler) GetTool(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "tool_name_required",
			"message": "Tool name is required",
		})
		return
	}

	tool := h.registry.GetTool(c.Request.Context(), name)
	if tool == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "tool_not_found",
			"message": "Tool not found: " + name,
		})
		return
	}

	// Marshal to JSON using protojson for proper proto message handling
	jsonBytes, err := protoMarshalOptions.Marshal(tool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to marshal tool",
			"message": err.Error(),
		})
		return
	}

	// Set cache headers (1 minute cache for discovery)
	c.Header("Cache-Control", "public, max-age=60")
	c.Header("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write(jsonBytes)
}
