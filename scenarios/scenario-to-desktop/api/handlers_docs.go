package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type DocsManifest struct {
	Version         string         `json:"version"`
	Title           string         `json:"title"`
	Description     string         `json:"description,omitempty"`
	DefaultDocument string         `json:"defaultDocument"`
	Sections        []DocsSection  `json:"sections"`
	Navigation      DocsNavigation `json:"navigation,omitempty"`
}

type DocsSection struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Icon        string         `json:"icon,omitempty"`
	Description string         `json:"description,omitempty"`
	Documents   []DocsDocument `json:"documents"`
}

type DocsNavigation struct {
	Primary   []string `json:"primary,omitempty"`
	Secondary []string `json:"secondary,omitempty"`
}

type DocsDocument struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type DocsContentResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// docsManifestHandler serves the docs manifest for the in-UI browser.
func (s *Server) docsManifestHandler(w http.ResponseWriter, r *http.Request) {
	docsDir := s.resolveDocsDir()
	manifestPath := filepath.Join(docsDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "docs manifest not found", http.StatusNotFound)
			return
		}
		s.logger.Error("failed to read docs manifest", "path", manifestPath, "error", err)
		http.Error(w, "failed to read docs manifest", http.StatusInternalServerError)
		return
	}

	var manifest DocsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		s.logger.Error("invalid docs manifest format", "path", manifestPath, "error", err)
		http.Error(w, "invalid docs manifest format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

// docsContentHandler returns the markdown content for a given doc path.
func (s *Server) docsContentHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	docsDir := s.resolveDocsDir()
	fullPath := filepath.Join(docsDir, cleanPath)

	absDocsDir, _ := filepath.Abs(docsDir)
	absFullPath, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(detectVrooliRoot())

	if !strings.HasPrefix(absFullPath, absDocsDir) && !strings.HasPrefix(absFullPath, absRoot) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "document not found", http.StatusNotFound)
			return
		}
		s.logger.Error("failed to read doc file", "path", fullPath, "error", err)
		http.Error(w, "failed to read document", http.StatusInternalServerError)
		return
	}

	response := DocsContentResponse{
		Path:    path,
		Content: string(data),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// resolveDocsDir finds the docs directory, preferring the scenario root.
func (s *Server) resolveDocsDir() string {
	if env := os.Getenv("SCENARIO_TO_DESKTOP_DOCS_DIR"); env != "" {
		return env
	}

	candidates := []string{
		filepath.Join(detectVrooliRoot(), "scenarios", "scenario-to-desktop", "docs"),
		"../docs",
		"./docs",
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if _, err := os.Stat(filepath.Join(candidate, "manifest.json")); err == nil {
				return candidate
			}
		}
	}

	return filepath.Join(detectVrooliRoot(), "scenarios", "scenario-to-desktop", "docs")
}
