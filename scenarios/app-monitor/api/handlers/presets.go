package handlers

import (
	"app-monitor-api/repository"
	"app-monitor-api/services"
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PresetHandler handles HTTP requests for workspace presets.
type PresetHandler struct {
	presetService *services.PresetService
}

// NewPresetHandler creates a new PresetHandler.
func NewPresetHandler(presetService *services.PresetService) *PresetHandler {
	return &PresetHandler{presetService: presetService}
}

// ListPresets returns all workspace presets.
func (h *PresetHandler) ListPresets(c *gin.Context) {
	HandleServiceCallRaw(c, func(ctx context.Context) ([]repository.WorkspacePreset, error) {
		return h.presetService.ListPresets(ctx)
	}, "Failed to list presets")
}

// GetPreset returns a single workspace preset by ID.
func (h *PresetHandler) GetPreset(c *gin.Context) {
	id := c.Param("id")
	HandleServiceCall(c, func(ctx context.Context) (*repository.WorkspacePreset, error) {
		return h.presetService.GetPreset(ctx, id)
	}, "Preset not found")
}

// CreatePreset creates a new workspace preset and returns it with 201 status.
func (h *PresetHandler) CreatePreset(c *gin.Context) {
	var preset repository.WorkspacePreset
	if err := c.ShouldBindJSON(&preset); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
		return
	}

	err := h.presetService.CreatePreset(c.Request.Context(), &preset)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrPresetNameRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrDatabaseUnavailable):
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, errorResponse("Failed to create preset"))
		return
	}

	c.JSON(http.StatusCreated, successResponse(&preset))
}

// UpdatePreset updates an existing workspace preset.
func (h *PresetHandler) UpdatePreset(c *gin.Context) {
	id := c.Param("id")
	var preset repository.WorkspacePreset
	if err := c.ShouldBindJSON(&preset); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
		return
	}
	preset.ID = id

	HandleServiceAction(c, func(ctx context.Context) error {
		return h.presetService.UpdatePreset(ctx, &preset)
	}, "Preset updated", "Failed to update preset")
}

// DeletePreset removes a workspace preset by ID.
func (h *PresetHandler) DeletePreset(c *gin.Context) {
	id := c.Param("id")
	HandleServiceAction(c, func(ctx context.Context) error {
		return h.presetService.DeletePreset(ctx, id)
	}, "Preset deleted", "Failed to delete preset")
}
