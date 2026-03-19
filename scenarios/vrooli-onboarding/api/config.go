package main

import (
	"fmt"
	"net/http"
)

// configGenerateRequest is the expected body for POST /api/v1/config/generate.
type configGenerateRequest struct {
	Resources []string `json:"resources"`
}

// serviceJSONSnippet represents a generated service.json fragment.
type serviceJSONSnippet struct {
	Resources map[string]resourceConfig `json:"resources"`
}

type resourceConfig struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
}

// loadAvailableSet loads resources and returns a set of available resource names.
func loadAvailableSet() (map[string]bool, error) {
	resources, err := loadResources()
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(resources))
	for _, res := range resources {
		set[res.Name] = true
	}
	return set, nil
}

func (s *Server) handleConfigGenerate(w http.ResponseWriter, r *http.Request) {
	var req configGenerateRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if len(req.Resources) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "resources list must not be empty",
		})
		return
	}

	availableSet, err := loadAvailableSet()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to load available resources: " + err.Error(),
		})
		return
	}

	snippet := serviceJSONSnippet{
		Resources: make(map[string]resourceConfig, len(req.Resources)),
	}
	var unknownResources []string

	for _, name := range req.Resources {
		if !availableSet[name] {
			unknownResources = append(unknownResources, name)
			continue
		}
		snippet.Resources[name] = resourceConfig{
			Enabled: true,
			Name:    name,
		}
	}

	resp := map[string]any{
		"config": snippet,
	}
	if len(unknownResources) > 0 {
		resp["warnings"] = []string{
			fmt.Sprintf("unknown resources skipped: %v", unknownResources),
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// configValidateRequest is the expected body for POST /api/v1/config/validate.
type configValidateRequest struct {
	Resources map[string]resourceConfig `json:"resources"`
}

// validationResult holds per-resource validation results.
type validationResult struct {
	Resource string   `json:"resource"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func (s *Server) handleConfigValidate(w http.ResponseWriter, r *http.Request) {
	var req configValidateRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if len(req.Resources) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "resources map must not be empty",
		})
		return
	}

	availableSet, err := loadAvailableSet()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to load available resources: " + err.Error(),
		})
		return
	}

	// Build set of enabled resources in the config
	enabledSet := make(map[string]bool, len(req.Resources))
	for name, cfg := range req.Resources {
		if cfg.Enabled {
			enabledSet[name] = true
		}
	}

	results := make([]validationResult, 0, len(req.Resources))
	allValid := true

	for name, cfg := range req.Resources {
		vr := validationResult{
			Resource: name,
			Valid:    true,
		}

		if !availableSet[name] {
			vr.Valid = false
			vr.Errors = append(vr.Errors, "resource not found in available resources")
			allValid = false
			results = append(results, vr)
			continue
		}

		if !cfg.Enabled {
			vr.Warnings = append(vr.Warnings, "resource is disabled in config")
		}

		// Check dependencies
		if deps, ok := knownDependencies[name]; ok && cfg.Enabled {
			for _, dep := range deps {
				if !enabledSet[dep] {
					vr.Warnings = append(vr.Warnings,
						fmt.Sprintf("dependency %q is not enabled in config", dep))
				}
			}
		}

		results = append(results, vr)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":   allValid,
		"results": results,
	})
}
