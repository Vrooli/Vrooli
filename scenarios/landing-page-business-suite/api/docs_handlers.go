package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	docshandler "landing-page-business-suite-api/handlers/docs"
	"landing-page-business-suite-api/internal/envx"
)

// DocEntry represents a documentation file or directory
type DocEntry = docshandler.Entry

// DocContent represents the content of a documentation file
type DocContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Title   string `json:"title"`
}

// getDocsRoot returns the absolute path to the docs directory
func getDocsRoot() string {
	// Check for explicit override first
	if override := strings.TrimSpace(envx.Get("DOCS_ROOT")); override != "" {
		if abs, err := filepath.Abs(override); err == nil {
			return abs
		}
		return override
	}

	// Check for scenario root
	if scenarioRoot := strings.TrimSpace(envx.Get("SCENARIO_ROOT")); scenarioRoot != "" {
		return filepath.Join(scenarioRoot, "docs")
	}

	// Fallback: API runs from api/ subdirectory, so go up one level
	// Always resolve to absolute path for consistent behavior
	if abs, err := filepath.Abs(filepath.Join("..", "docs")); err == nil {
		return abs
	}
	return filepath.Join("..", "docs")
}

// handleDocsTree returns the hierarchical structure of docs files
func handleDocsTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docsRoot := getDocsRoot()

		// Resolve to absolute path for logging
		absPath, _ := filepath.Abs(docsRoot)
		logStructured("docs_tree_request", map[string]interface{}{
			"docs_root":     docsRoot,
			"absolute_path": absPath,
		})

		// Check if docs directory exists
		if _, err := os.Stat(docsRoot); os.IsNotExist(err) {
			logStructured("docs_directory_not_found", map[string]interface{}{
				"path":  docsRoot,
				"error": err.Error(),
			})
			writeJSONSuccessData(w, []DocEntry{})
			return
		}

		entries, err := buildDocsTree(docsRoot, "")
		if err != nil {
			logStructuredError("docs_tree_build_failed", map[string]interface{}{
				"path":  docsRoot,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to read docs directory", ApiErrorTypeServerError)
			return
		}

		logStructured("docs_tree_success", map[string]interface{}{
			"path":        docsRoot,
			"entry_count": len(entries),
		})

		writeJSONSuccessData(w, entries)
	}
}

// handleDocsContent returns the content of a specific doc file
func handleDocsContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docPath, ok := requireQueryParam(w, r, "path")
		if !ok {
			return
		}

		// Sanitize path to prevent directory traversal
		cleanPath := filepath.Clean(docPath)
		if strings.Contains(cleanPath, "..") {
			writeJSONError(w, http.StatusBadRequest, "Invalid path", ApiErrorTypeValidation)
			return
		}

		// Only allow .md files
		if !strings.HasSuffix(strings.ToLower(cleanPath), ".md") {
			writeJSONError(w, http.StatusBadRequest, "Only markdown files are allowed", ApiErrorTypeValidation)
			return
		}

		docsRoot := getDocsRoot()
		fullPath := filepath.Join(docsRoot, cleanPath)

		// Resolve to absolute for security comparison
		absFullPath, err := filepath.Abs(fullPath)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid path", ApiErrorTypeValidation)
			return
		}
		absDocsRoot, _ := filepath.Abs(docsRoot)

		// Verify the file is within docs directory. String-prefix checks are
		// insufficient because a sibling such as docs-backup shares the prefix.
		if !isWithinDirectory(absDocsRoot, absFullPath) {
			writeJSONError(w, http.StatusBadRequest, "Invalid path", ApiErrorTypeValidation)
			return
		}

		// #nosec G304 -- isWithinDirectory validates fullPath against docsRoot.
		content, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				writeJSONError(w, http.StatusNotFound, "File not found", ApiErrorTypeNotFound)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to read file", ApiErrorTypeServerError)
			return
		}

		// Extract title from first H1 heading or filename
		title := extractTitle(string(content), cleanPath)

		doc := DocContent{
			Path:    cleanPath,
			Content: string(content),
			Title:   title,
		}

		writeJSONSuccessData(w, doc)
	}
}

func isWithinDirectory(root, path string) bool {
	return docshandler.IsWithinDirectory(root, path)
}

// buildDocsTree recursively builds the docs file tree
func buildDocsTree(root, relativePath string) ([]DocEntry, error) {
	return docshandler.BuildTree(root, relativePath)
}

// extractTitle extracts the title from markdown content or filename
func extractTitle(content, path string) string {
	return docshandler.ExtractTitle(content, path)
}
