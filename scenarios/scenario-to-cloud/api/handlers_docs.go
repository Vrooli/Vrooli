package main

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

// handleGetDocsManifest serves the docs manifest.json file
func (s *Server) handleGetDocsManifest(w http.ResponseWriter, r *http.Request) {
	docsDir := s.getDocsDir()
	manifestPath := filepath.Join(docsDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeAPIError(w, http.StatusNotFound, APIError{
				Code:    "docs_not_found",
				Message: "Documentation manifest not found",
				Hint:    "Ensure docs/manifest.json exists in the scenario directory",
			})
			return
		}
		s.log("failed to read docs manifest", map[string]interface{}{"error": err.Error()})
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "read_error",
			Message: "Failed to read documentation manifest",
		})
		return
	}

	var manifest DocsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		s.log("failed to parse docs manifest", map[string]interface{}{"error": err.Error()})
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "parse_error",
			Message: "Invalid documentation manifest format",
		})
		return
	}

	writeJSON(w, http.StatusOK, manifest)
}

// handleGetDocContent serves the content of a specific doc file
func (s *Server) handleGetDocContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_path",
			Message: "Path parameter is required",
			Hint:    "Provide ?path=QUICKSTART.md",
		})
		return
	}

	// Prevent directory traversal attacks
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || containsParentRef(cleanPath) {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_path",
			Message: "Invalid document path",
			Hint:    "Path cannot be absolute or contain parent references",
		})
		return
	}

	docsDir := s.getDocsDir()
	fullPath := filepath.Join(docsDir, cleanPath)

	// Verify the path is still within docs directory
	absDocsDir, _ := filepath.Abs(docsDir)
	absFullPath, _ := filepath.Abs(fullPath)
	if len(absFullPath) < len(absDocsDir) || absFullPath[:len(absDocsDir)] != absDocsDir {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_path",
			Message: "Invalid document path",
		})
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeAPIError(w, http.StatusNotFound, APIError{
				Code:    "doc_not_found",
				Message: "Document not found",
				Hint:    "Check that the path matches a document in the manifest",
			})
			return
		}
		s.log("failed to read doc file", map[string]interface{}{
			"path":  path,
			"error": err.Error(),
		})
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "read_error",
			Message: "Failed to read document",
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
func (s *Server) getDocsDir() string {
	// Check for environment variable override
	if dir := os.Getenv("SCENARIO_TO_CLOUD_DOCS_DIR"); dir != "" {
		return dir
	}

	// Look for docs directory relative to typical locations
	candidates := []string{
		"docs",
		"../docs",
		"../../docs",
		filepath.Join(os.Getenv("HOME"), "Vrooli/scenarios/scenario-to-cloud/docs"),
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
	// Check for .. anywhere in the path
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	// Also check for .. at start
	return strings.HasPrefix(path, "..")
}
