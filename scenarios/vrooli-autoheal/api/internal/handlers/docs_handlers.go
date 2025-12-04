package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DocsManifest represents the structure of the docs manifest.json file
type DocsManifest struct {
	Version         string        `json:"version"`
	Title           string        `json:"title"`
	Description     string        `json:"description,omitempty"`
	DefaultDocument string        `json:"defaultDocument"`
	Sections        []DocsSection `json:"sections"`
}

// DocsSection represents a section in the docs manifest
type DocsSection struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Icon        string         `json:"icon,omitempty"`
	Description string         `json:"description,omitempty"`
	Visibility  string         `json:"visibility,omitempty"`
	Documents   []DocsDocument `json:"documents"`
}

// DocsDocument represents a document entry in the manifest
type DocsDocument struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// DocsContentResponse wraps the markdown content response
type DocsContentResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// DocsManifest serves the docs manifest.json file
func (h *Handlers) DocsManifest(w http.ResponseWriter, r *http.Request) {
	docsDir := h.getDocsDir()
	manifestPath := filepath.Join(docsDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "docs manifest not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to read docs manifest",
		})
		return
	}

	var manifest DocsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "invalid docs manifest format",
		})
		return
	}

	writeJSON(w, http.StatusOK, manifest)
}

// DocsContent serves the content of a specific doc file
func (h *Handlers) DocsContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "path parameter is required",
		})
		return
	}

	// Prevent directory traversal attacks
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || containsParentRef(cleanPath) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid path",
		})
		return
	}

	docsDir := h.getDocsDir()
	fullPath := filepath.Join(docsDir, cleanPath)

	// Verify the path is still within docs directory
	absDocsDir, err := filepath.Abs(docsDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to resolve docs directory",
		})
		return
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to resolve document path",
		})
		return
	}

	if !strings.HasPrefix(absFullPath, absDocsDir) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid path",
		})
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "document not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to read document",
		})
		return
	}

	response := DocsContentResponse{
		Path:    path,
		Content: string(data),
	}

	writeJSON(w, http.StatusOK, response)
}

// getDocsDir returns the path to the docs directory
func (h *Handlers) getDocsDir() string {
	// Check for environment variable override
	if dir := os.Getenv("AUTOHEAL_DOCS_DIR"); dir != "" {
		return dir
	}

	// Default to docs relative to working directory
	// or look for it relative to the scenario root
	candidates := []string{
		"docs",
		"../docs",
		"../../docs",
		filepath.Join(os.Getenv("HOME"), "Vrooli/scenarios/vrooli-autoheal/docs"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			manifestPath := filepath.Join(candidate, "manifest.json")
			if _, err := os.Stat(manifestPath); err == nil {
				return candidate
			}
		}
	}

	// Return default even if not found - handlers will return 404
	return "docs"
}

// containsParentRef checks if a path contains parent directory references
func containsParentRef(path string) bool {
	// Check for .. at start or after separator
	if strings.HasPrefix(path, "..") {
		return true
	}
	if strings.Contains(path, string(filepath.Separator)+"..") {
		return true
	}
	// Also check for forward slash on all platforms
	if strings.Contains(path, "/..") {
		return true
	}
	return false
}

// writeJSON is a helper to write JSON responses with proper headers and status
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
