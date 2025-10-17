package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// CatalogEntry represents a scenario or resource with its PRD status
type CatalogEntry struct {
	Type        string `json:"type"`        // "scenario" or "resource"
	Name        string `json:"name"`        // Entity name
	HasPRD      bool   `json:"has_prd"`     // Whether PRD.md exists
	PRDPath     string `json:"prd_path"`    // Absolute path to PRD.md
	HasDraft    bool   `json:"has_draft"`   // Whether a draft exists
	Description string `json:"description"` // Brief description (if available)
}

// CatalogResponse represents the catalog API response
type CatalogResponse struct {
	Entries []CatalogEntry `json:"entries"`
	Total   int            `json:"total"`
}

// PublishedPRDResponse represents a published PRD content
type PublishedPRDResponse struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Path    string `json:"path"`
}

func handleGetCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	entries := []CatalogEntry{}

	// Enumerate scenarios
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")
	scenarios, err := enumerateEntities(scenariosDir, "scenario")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enumerate scenarios: %v", err), http.StatusInternalServerError)
		return
	}
	entries = append(entries, scenarios...)

	// Enumerate resources
	resourcesDir := filepath.Join(vrooliRoot, "resources")
	resources, err := enumerateEntities(resourcesDir, "resource")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enumerate resources: %v", err), http.StatusInternalServerError)
		return
	}
	entries = append(entries, resources...)

	// Check for drafts
	for i := range entries {
		entries[i].HasDraft = hasDraft(entries[i].Type, entries[i].Name)
	}

	response := CatalogResponse{
		Entries: entries,
		Total:   len(entries),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func enumerateEntities(baseDir string, entityType string) ([]CatalogEntry, error) {
	entries := []CatalogEntry{}

	// Check if directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return entries, nil // Return empty list if directory doesn't exist
	}

	// Read directory entries
	dirEntries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", baseDir, err)
	}

	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}

		name := dirEntry.Name()
		prdPath := filepath.Join(baseDir, name, "PRD.md")

		// Check if PRD.md exists
		hasPRD := false
		if _, err := os.Stat(prdPath); err == nil {
			hasPRD = true
		}

		entry := CatalogEntry{
			Type:    entityType,
			Name:    name,
			HasPRD:  hasPRD,
			PRDPath: prdPath,
		}

		// Extract brief description from PRD if available
		if hasPRD {
			description := extractDescription(prdPath)
			entry.Description = description
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func extractDescription(prdPath string) string {
	content, err := os.ReadFile(prdPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")

	// Look for the first non-empty line after the title
	foundTitle := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip markdown header
		if strings.HasPrefix(line, "#") {
			foundTitle = true
			continue
		}

		// Return first substantial line after title
		if foundTitle && len(line) > 0 && !strings.HasPrefix(line, "##") {
			// Remove markdown formatting
			line = strings.TrimPrefix(line, "> ")
			line = strings.TrimPrefix(line, "**")
			line = strings.TrimSuffix(line, "**")

			// Truncate if too long
			if len(line) > 200 {
				line = line[:197] + "..."
			}

			return line
		}
	}

	return ""
}

func hasDraft(entityType string, entityName string) bool {
	draftPath := getDraftPath(entityType, entityName)
	_, err := os.Stat(draftPath)
	return err == nil
}

func getDraftPath(entityType string, entityName string) string {
	return filepath.Join("../data/prd-drafts", entityType, entityName+".md")
}

func handleGetPublishedPRD(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	// Validate entity type
	if entityType != "scenario" && entityType != "resource" {
		http.Error(w, "Invalid entity type. Must be 'scenario' or 'resource'", http.StatusBadRequest)
		return
	}

	// Construct PRD path
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseDir := filepath.Join(vrooliRoot, entityType+"s")
	prdPath := filepath.Join(baseDir, entityName, "PRD.md")

	// Read PRD content
	content, err := os.ReadFile(prdPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("PRD not found for %s/%s", entityType, entityName), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read PRD: %v", err), http.StatusInternalServerError)
		return
	}

	response := PublishedPRDResponse{
		Type:    entityType,
		Name:    entityName,
		Content: string(content),
		Path:    prdPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
