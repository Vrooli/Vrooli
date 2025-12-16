package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/gorilla/mux"
)

type templateVarJSON struct {
	Flag        string `json:"flag"`
	Description string `json:"description"`
	Default     string `json:"default"`
}

type templateManifestJSON struct {
	Name         string                     `json:"name"`
	DisplayName  string                     `json:"displayName"`
	Description  string                     `json:"description"`
	Stack        []string                   `json:"stack"`
	RequiredVars map[string]templateVarJSON `json:"requiredVars"`
	OptionalVars map[string]templateVarJSON `json:"optionalVars"`
}

type templateVarDefinition struct {
	Name        string
	Flag        string
	Description string
	Default     string
	Required    bool
}

type TemplateManifest struct {
	Name         string
	DisplayName  string
	Description  string
	Stack        []string
	BasePath     string
	RequiredVars map[string]templateVarDefinition
	OptionalVars map[string]templateVarDefinition
}

type ScenarioTemplateVar struct {
	Name        string `json:"name"`
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
}

type ScenarioTemplateResponse struct {
	Name        string                `json:"name"`
	DisplayName string                `json:"display_name"`
	Description string                `json:"description"`
	Stack       []string              `json:"stack"`
	Required    []ScenarioTemplateVar `json:"required_vars"`
	Optional    []ScenarioTemplateVar `json:"optional_vars"`
}

type ScenarioTemplateListResponse struct {
	Templates []ScenarioTemplateResponse `json:"templates"`
}

func handleListScenarioTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := listScenarioTemplates()
	if err != nil {
		respondInternalError(w, "Failed to load scenario templates", err)
		return
	}

	respondJSON(w, http.StatusOK, ScenarioTemplateListResponse{Templates: templates})
}

func handleGetScenarioTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	if strings.TrimSpace(name) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "template name is required"})
		return
	}

	manifest, err := loadScenarioTemplateManifest(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		respondInternalError(w, "Failed to load template", err)
		return
	}

	respondJSON(w, http.StatusOK, manifestToResponse(manifest))
}

func listScenarioTemplates() ([]ScenarioTemplateResponse, error) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return nil, err
	}

	templatesDir := filepath.Join(vrooliRoot, "scripts", "scenarios", "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	responses := make([]ScenarioTemplateResponse, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifest, err := loadScenarioTemplateManifest(entry.Name())
		if err != nil {
			slog.Warn("Failed to load scenario template", "template", entry.Name(), "error", err)
			continue
		}
		responses = append(responses, manifestToResponse(manifest))
	}

	sort.Slice(responses, func(i, j int) bool {
		return responses[i].DisplayName < responses[j].DisplayName
	})

	return responses, nil
}

func loadScenarioTemplateManifest(name string) (*TemplateManifest, error) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(vrooliRoot, "scripts", "scenarios", "templates", name, "template.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var raw templateManifestJSON
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse template manifest: %w", err)
	}

	manifest := &TemplateManifest{
		Name:         raw.Name,
		DisplayName:  raw.DisplayName,
		Description:  raw.Description,
		Stack:        raw.Stack,
		BasePath:     filepath.Dir(manifestPath),
		RequiredVars: make(map[string]templateVarDefinition),
		OptionalVars: make(map[string]templateVarDefinition),
	}

	if manifest.Name == "" {
		manifest.Name = name
	}
	if manifest.DisplayName == "" {
		manifest.DisplayName = titleFromSlug(name)
	}

	for key, value := range raw.RequiredVars {
		manifest.RequiredVars[key] = templateVarDefinition{
			Name:        key,
			Flag:        value.Flag,
			Description: value.Description,
			Default:     value.Default,
			Required:    true,
		}
	}

	for key, value := range raw.OptionalVars {
		manifest.OptionalVars[key] = templateVarDefinition{
			Name:        key,
			Flag:        value.Flag,
			Description: value.Description,
			Default:     value.Default,
			Required:    false,
		}
	}

	return manifest, nil
}

func titleFromSlug(slug string) string {
	slug = strings.ReplaceAll(strings.TrimSpace(slug), "-", " ")
	words := strings.Fields(slug)
	for i, word := range words {
		lowered := []rune(strings.ToLower(word))
		if len(lowered) == 0 {
			continue
		}
		lowered[0] = unicode.ToUpper(lowered[0])
		words[i] = string(lowered)
	}
	return strings.Join(words, " ")
}

func manifestToResponse(manifest *TemplateManifest) ScenarioTemplateResponse {
	return ScenarioTemplateResponse{
		Name:        manifest.Name,
		DisplayName: manifest.DisplayName,
		Description: manifest.Description,
		Stack:       manifest.Stack,
		Required:    buildVarList(manifest.RequiredVars),
		Optional:    buildVarList(manifest.OptionalVars),
	}
}

func buildVarList(vars map[string]templateVarDefinition) []ScenarioTemplateVar {
	if len(vars) == 0 {
		return nil
	}

	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	list := make([]ScenarioTemplateVar, 0, len(keys))
	for _, key := range keys {
		def := vars[key]
		list = append(list, ScenarioTemplateVar{
			Name:        def.Name,
			Flag:        def.Flag,
			Description: def.Description,
			Default:     def.Default,
			Required:    def.Required,
		})
	}

	return list
}
