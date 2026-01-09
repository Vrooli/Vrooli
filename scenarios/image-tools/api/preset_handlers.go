package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// handleListPresets returns all available preset profiles
func (s *Server) handleListPresets(c *fiber.Ctx) error {
	presets := ListPresets()
	return c.JSON(fiber.Map{
		"presets": presets,
		"count":   len(presets),
	})
}

// handleGetPreset returns a specific preset by name
func (s *Server) handleGetPreset(c *fiber.Ctx) error {
	name := c.Params("name")
	preset, ok := GetPreset(name)
	if !ok {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Preset not found",
			"available_presets": func() []string {
				names := make([]string, 0, len(BuiltinPresets))
				for k := range BuiltinPresets {
					names = append(names, k)
				}
				return names
			}(),
		})
	}
	return c.JSON(preset)
}

// handleApplyPreset applies a preset profile to an uploaded image
// This is a simplified version that returns the preset configuration
// The actual processing should be done by calling the individual endpoints
func (s *Server) handleApplyPreset(c *fiber.Ctx) error {
	presetName := c.Params("name")
	preset, ok := GetPreset(presetName)
	if !ok {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Preset not found",
			"available_presets": func() []string {
				names := make([]string, 0, len(BuiltinPresets))
				for k := range BuiltinPresets {
					names = append(names, k)
				}
				return names
			}(),
		})
	}

	// Return the preset configuration for client-side or batch processing
	return c.JSON(fiber.Map{
		"preset":      preset,
		"message":     "Use the /batch endpoint with these operations to apply the preset",
		"batch_ready": true,
		"operations":  preset.Operations,
	})
}
