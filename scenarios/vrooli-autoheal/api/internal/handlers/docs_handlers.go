package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	apierrors "vrooli-autoheal/internal/errors"
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
			apierrors.LogAndRespond(w, apierrors.NewNotFoundError("docs", "docs manifest", "manifest.json"))
			return
		}
		apierrors.LogAndRespond(w, apierrors.NewInternalError("docs", "Failed to read docs manifest", err))
		return
	}

	var manifest DocsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewInternalError("docs", "Invalid docs manifest format", err))
		return
	}

	writeJSON(w, http.StatusOK, manifest)
}

// DocsContent serves the content of a specific doc file
func (h *Handlers) DocsContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("docs", "path parameter is required", nil))
		return
	}

	// Prevent directory traversal attacks
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || containsParentRef(cleanPath) {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("docs", "path contains invalid characters", nil))
		return
	}

	docsDir := h.getDocsDir()
	fullPath := filepath.Join(docsDir, cleanPath)

	// Verify the path is still within docs directory
	absDocsDir, err := filepath.Abs(docsDir)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewInternalError("docs", "Failed to resolve docs directory", err))
		return
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewInternalError("docs", "Failed to resolve document path", err))
		return
	}

	if !strings.HasPrefix(absFullPath, absDocsDir) {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("docs", "path contains invalid characters", nil))
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			apierrors.LogAndRespond(w, apierrors.NewNotFoundError("docs", "document", cleanPath))
			return
		}
		apierrors.LogAndRespond(w, apierrors.NewInternalError("docs", "Failed to read document", err))
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

// writeJSON is a helper to write JSON responses with proper headers and status.
// Only used for success responses; errors go through apierrors.LogAndRespond.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		apierrors.LogError("docs", "encode_response", err)
	}
}
