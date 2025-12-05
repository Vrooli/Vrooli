package httpserver

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// DocsManifest represents the structure of the docs manifest.json file
type DocsManifest struct {
	Version         string        `json:"version"`
	Title           string        `json:"title"`
	DefaultDocument string        `json:"defaultDocument"`
	Sections        []DocsSection `json:"sections"`
}

// DocsSection represents a section in the docs manifest
type DocsSection struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	Visibility string         `json:"visibility,omitempty"`
	Documents  []DocsDocument `json:"documents"`
}

// DocsDocument represents a document entry in the manifest
type DocsDocument struct {
	Path  string `json:"path"`
	Title string `json:"title"`
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
			s.writeError(w, http.StatusNotFound, "docs manifest not found")
			return
		}
		s.log("failed to read docs manifest", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to read docs manifest")
		return
	}

	var manifest DocsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		s.log("failed to parse docs manifest", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "invalid docs manifest format")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

// handleGetDocContent serves the content of a specific doc file
func (s *Server) handleGetDocContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		s.writeError(w, http.StatusBadRequest, "path parameter is required")
		return
	}

	// Prevent directory traversal attacks
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || containsParentRef(cleanPath) {
		s.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	docsDir := s.getDocsDir()
	fullPath := filepath.Join(docsDir, cleanPath)

	// Verify the path is still within docs directory
	absDocsDir, _ := filepath.Abs(docsDir)
	absFullPath, _ := filepath.Abs(fullPath)
	if len(absFullPath) < len(absDocsDir) || absFullPath[:len(absDocsDir)] != absDocsDir {
		s.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.writeError(w, http.StatusNotFound, "document not found")
			return
		}
		s.log("failed to read doc file", map[string]interface{}{
			"path":  path,
			"error": err.Error(),
		})
		s.writeError(w, http.StatusInternalServerError, "failed to read document")
		return
	}

	response := DocsContentResponse{
		Path:    path,
		Content: string(data),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getDocsDir returns the path to the docs directory
func (s *Server) getDocsDir() string {
	// Check for environment variable override
	if dir := os.Getenv("TEST_GENIE_DOCS_DIR"); dir != "" {
		return dir
	}

	// Default to scenarios/test-genie/docs relative to working directory
	// or look for it relative to the scenario root
	candidates := []string{
		"docs",
		"../docs",
		"../../docs",
		filepath.Join(os.Getenv("HOME"), "Vrooli/scenarios/test-genie/docs"),
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
	for _, part := range filepath.SplitList(path) {
		if part == ".." {
			return true
		}
	}
	// Also check for .. in the path itself
	return len(path) >= 2 && path[0:2] == ".." ||
		len(path) >= 3 && (path[0:3] == "../" || path[0:3] == "..\\")
}
