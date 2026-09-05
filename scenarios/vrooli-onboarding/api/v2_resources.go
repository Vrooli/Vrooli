package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type resourceReadModel struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Enabled     bool   `json:"enabled"`
	Installed   bool   `json:"installed"`
}

type resourceGroupsResponse struct {
	Resources  []resourceReadModel `json:"resources"`
	Required   []resourceReadModel `json:"required"`
	Optional   []resourceReadModel `json:"optional"`
	Standalone []resourceReadModel `json:"standalone"`
}

// loadResourceReadModels reads only immutable resource declarations. Runtime
// health remains owned by the v1 control-plane endpoint; this read model must
// also work in a desktop bundle where the CLI is intentionally absent.
func loadResourceReadModels() ([]resourceReadModel, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	state, err := loadOperatorState()
	if err != nil {
		return nil, err
	}
	resourcesRoot := filepath.Join(root, "resources")
	entries, err := os.ReadDir(resourcesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &catalogUnavailableError{Missing: "catalog/resources", Remediation: "rebuild the bundle with the resource catalog, then restart onboarding"}
		}
		return nil, err
	}
	models := make([]resourceReadModel, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(resourcesRoot, entry.Name(), "resource.json")
		data, readErr := os.ReadFile(path)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, readErr
		}
		var manifest struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			Description string `json:"description"`
			Category    string `json:"category"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		name := strings.TrimSpace(manifest.Name)
		if name == "" {
			name = entry.Name()
		}
		choice := state.Resources[name]
		enabled := false
		if choice.Enabled != nil {
			enabled = *choice.Enabled
		}
		models = append(models, resourceReadModel{
			Name: name, DisplayName: manifest.DisplayName, Description: manifest.Description,
			Category: manifest.Category, Enabled: enabled, Installed: true,
		})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Name < models[j].Name })
	return models, nil
}

func (s *Server) handleV2Resources(w http.ResponseWriter, _ *http.Request) {
	models, err := loadResourceReadModels()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	root, err := manifestRoot()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	scenarioModels, err := loadScenarioReadModels()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	closure, err := resolveClosure(root, scenarioModels)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	byName := make(map[string]resourceReadModel, len(models))
	for _, model := range models {
		byName[model.Name] = model
	}
	response := resourceGroupsResponse{Resources: models, Required: []resourceReadModel{}, Optional: []resourceReadModel{}, Standalone: []resourceReadModel{}}
	for _, member := range closure.Resources {
		model, ok := byName[member.Name]
		if !ok {
			continue
		}
		if member.Required {
			model.Enabled = true
			response.Required = append(response.Required, model)
		} else {
			response.Optional = append(response.Optional, model)
		}
		delete(byName, member.Name)
	}
	for _, model := range byName {
		response.Standalone = append(response.Standalone, model)
	}
	sort.Slice(response.Required, func(i, j int) bool { return response.Required[i].Name < response.Required[j].Name })
	sort.Slice(response.Optional, func(i, j int) bool { return response.Optional[i].Name < response.Optional[j].Name })
	sort.Slice(response.Standalone, func(i, j int) bool { return response.Standalone[i].Name < response.Standalone[j].Name })
	writeJSON(w, http.StatusOK, map[string]any{
		"resources":  response.Resources,
		"required":   response.Required,
		"optional":   response.Optional,
		"standalone": response.Standalone,
		"count":      len(response.Resources),
	})
}
